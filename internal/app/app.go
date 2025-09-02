package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	botpkg "github.com/NastyaGoryachaya/crypto-rate-service/internal/bot"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/bot/adapter"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/config"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/infra/stockapi"
	repopg "github.com/NastyaGoryachaya/crypto-rate-service/internal/repository/postgres"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/scheduler"
	fetchsvc "github.com/NastyaGoryachaya/crypto-rate-service/internal/service/fetch"
	ratesvc "github.com/NastyaGoryachaya/crypto-rate-service/internal/service/rates"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/transport/httptransport"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

type App struct {
	cfg config.Config
	log *slog.Logger

	db   *pgxpool.Pool
	e    *echo.Echo
	serv *http.Server

	coinRepo  *repopg.CoinRepo
	priceRepo *repopg.PriceRepo
	subsRepo  *repopg.SubscriptionRepo

	rates ratesvc.Service
	fetch fetchsvc.Service

	updater *scheduler.Scheduler

	bot *botpkg.Bot
}

func NewApp(cfg config.Config, log *slog.Logger, db *pgxpool.Pool) (*App, error) {
	app := &App{cfg: cfg, log: log, db: db}

	app.coinRepo = repopg.NewCoinRepository(db)
	app.priceRepo = repopg.NewPriceRepository(db)
	app.subsRepo = repopg.NewSubscriptionRepository(db)

	e := echo.New()
	app.e = e

	provider := stockapi.NewClient(stockapi.Config{
		BaseURL:   cfg.CoinGecko.BaseURL,
		Coins:     cfg.CoinGecko.Coins,
		Currency:  cfg.CoinGecko.Currency,
		Timeout:   cfg.CoinGecko.Timeout,
		UserAgent: cfg.CoinGecko.UserAgent,
	})

	app.rates = ratesvc.NewService(app.coinRepo, app.priceRepo, log)
	app.fetch = fetchsvc.NewService(provider, app.coinRepo, app.priceRepo, log)

	rh := httptransport.NewRatesHandler(log, app.rates, cfg.Server.ReadTimeout)
	rh.RegisterRoutes(e)

	app.serv = &http.Server{
		Addr:         cfg.Server.Addr,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
		Handler:      e,
	}

	if cfg.Scheduler.Enabled {
		app.updater = scheduler.NewScheduler(app.fetch, cfg.Scheduler.Interval, log)
	}

	if cfg.Telegram.Enabled {
		// Если бот включён, отсутствие токена — ошибка конфигурации
		token := strings.TrimSpace(cfg.Telegram.Token)
		if token == "" {
			log.Error("telegram enabled but TELEGRAM_BOT_TOKEN is empty")
			return nil, errors.New("telegram token is empty")
		}

		botApp, err := botpkg.New(
			botpkg.Config{Token: token, LongPollTimeout: 10 * time.Second},
			adapter.NewRatesReader(app.rates),
			app.subsRepo,
			log,
		)
		if err != nil {
			log.Error("telegram init failed", slog.String("error", err.Error()))
			return nil, err
		}
		app.bot = botApp
	}
	log.Info("app initialized",
		slog.Bool("telegram_enabled", cfg.Telegram.Enabled),
		slog.Bool("bot_attached", app.bot != nil),
		slog.String("http_addr", cfg.Server.Addr),
	)
	return app, nil
}

func (a *App) Run(ctx context.Context) error {
	if a.updater != nil {
		a.log.Info("starting updater")
		go a.updater.Start(ctx)
	}

	if a.bot != nil {
		a.log.Info("starting bot")
		go a.bot.Start(ctx)
	}

	a.log.Info("starting server", slog.String("addr", a.cfg.Server.Addr))
	go func() {
		if err := a.e.StartServer(a.serv); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.log.Error("http server error", slog.String("error", err.Error()))
		}
	}()
	<-ctx.Done()
	return a.Shutdown(context.Background())
}

func (a *App) Shutdown(ctx context.Context) error {
	shCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if a.e != nil {
		if err := a.e.Shutdown(shCtx); err != nil {
			a.log.Error("http shutdown error", slog.String("error", err.Error()))
		}
	}

	if a.bot != nil {
		a.bot.Stop()
	}

	if a.db != nil {
		a.db.Close()
	}

	a.log.Info("application stopped")
	return nil
}
