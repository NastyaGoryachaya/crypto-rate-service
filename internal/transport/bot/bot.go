package bot

import (
	"context"
	"log/slog"
	"time"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/config"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/interfaces"
	"gopkg.in/telebot.v4"
)

// Bot — основной тип приложения
type Bot struct {
	bot       *telebot.Bot
	svc       interfaces.Service
	subs      interfaces.SubscriptionCommander
	scheduler *scheduler
	logger    *slog.Logger
}

// New создаёт новый экземпляр приложения
func New(cfg config.TelegramConfig, svc interfaces.Service, subs interfaces.SubscriptionCommander, logger *slog.Logger, scheduler *scheduler) (*Bot, error) {
	const defaultPollTimeout = 10 * time.Second

	b, err := telebot.NewBot(telebot.Settings{
		Token:  cfg.Token,
		Poller: &telebot.LongPoller{Timeout: defaultPollTimeout},
	})
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		bot:       b,
		svc:       svc,
		subs:      subs,
		logger:    logger,
		scheduler: scheduler,
	}

	// маршруты команд
	b.Handle("/start", bot.handleStart)
	b.Handle("/rates", bot.handleRates)
	b.Handle("/startauto", bot.handleStartAuto)
	b.Handle("/stopauto", bot.handleStopAuto)
	return bot, nil
}

// Start запускает бота и планировщик
func (b *Bot) Start(ctx context.Context) {
	if b.scheduler != nil {
		go b.scheduler.run(ctx)
	}
	go b.bot.Start()
}

// Stop останавливает бота
func (b *Bot) Stop() {
	b.bot.Stop()
}
