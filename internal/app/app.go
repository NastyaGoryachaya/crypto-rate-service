package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/config"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/infra/api_client"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/infra/db"
	repopg "github.com/NastyaGoryachaya/crypto-rate-service/internal/repository/postgres"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/schedulers/scheduler_dispatcher"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/schedulers/scheduler_fetcher"
	ratesvc "github.com/NastyaGoryachaya/crypto-rate-service/internal/service/rates"
	subsvc "github.com/NastyaGoryachaya/crypto-rate-service/internal/service/subscription"
	botpkg "github.com/NastyaGoryachaya/crypto-rate-service/internal/transport/bot"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/transport/web"
	"github.com/NastyaGoryachaya/crypto-rate-service/pkg/logger"
	"github.com/labstack/echo/v4"
	"gopkg.in/telebot.v4"
)

func Run() error {
	// context + signals
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// config
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	// logger
	appLog := logger.New(&cfg.Logger)
	appLog.Info("starting crypto-rate-service")

	// db
	pool, err := db.NewPool(&cfg.Postgres)
	if err != nil {
		appLog.Error("db connect failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// repo
	coinRepo := repopg.NewCoinRepo(pool)
	subsRepo := repopg.NewSubscriptionRepo(pool)

	// client for API CoinGecko
	provider := api_client.NewClient(config.CoinGeckoConfig{
		BaseURL:   cfg.CoinGecko.BaseURL,
		Coins:     cfg.CoinGecko.Coins,
		Currency:  cfg.CoinGecko.Currency,
		Timeout:   cfg.CoinGecko.Timeout,
		UserAgent: cfg.CoinGecko.UserAgent,
	})

	// services
	ratesSvc := ratesvc.NewService(coinRepo, provider, appLog)

	// subscription service (бот)
	tbot, err := telebot.NewBot(telebot.Settings{
		Token:  cfg.Telegram.Token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		return err
	}
	subsSvc := subsvc.New(tbot, subsRepo, provider, appLog)

	// http
	e := echo.New()
	rh := web.NewRatesHandler(appLog, ratesSvc, cfg.Server.ReadTimeout)
	rh.RegisterRoutes(e)

	serv := &http.Server{
		Addr:         cfg.Server.Addr,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
		Handler:      e,
	}

	// schedulers
	var updater *scheduler_fetcher.Scheduler
	if cfg.Scheduler.Enabled {
		updater = scheduler_fetcher.NewScheduler(ratesSvc, cfg.Scheduler.Interval, appLog)
	}

	// telegram bot
	var bot *botpkg.Bot
	if cfg.Telegram.Enabled {
		token := strings.TrimSpace(cfg.Telegram.Token)
		if token == "" {
			return errors.New("telegram enabled but TELEGRAM_BOT_TOKEN is empty")
		}
		botSched := scheduler_dispatcher.NewScheduler(subsSvc, cfg.Scheduler.Interval, appLog)
		bot, err = botpkg.New(
			tbot,
			ratesSvc,
			subsSvc, // implements SubscriptionCommander
			appLog,
			botSched,
		)
		if err != nil {
			return err
		}
	}
	// starting goroutines
	if updater != nil {
		appLog.Info("starting updater")
		go updater.Start(ctx)
	}

	if bot != nil {
		appLog.Info("starting subscription bot")
		go bot.Start(ctx)
	}

	appLog.Info("starting http server", slog.String("addr", cfg.Server.Addr))
	go func() {
		if err := e.StartServer(serv); err != nil && !errors.Is(err, http.ErrServerClosed) {
			appLog.Error("http server error", slog.String("error", err.Error()))
		}
	}()

	// wait stop
	<-ctx.Done()

	// graceful shutdown
	shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if e != nil {
		_ = e.Shutdown(shCtx)
	}
	if bot != nil {
		bot.Stop()
	}

	appLog.Info("crypto-rate-service stopped")
	return nil
}
