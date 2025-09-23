package bot

import (
	"context"
	"log/slog"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/interfaces"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/schedulers/scheduler_dispatcher"
	"gopkg.in/telebot.v4"
)

// Bot — основной тип приложения
type Bot struct {
	bot       *telebot.Bot
	svc       interfaces.Service
	subs      interfaces.SubscriptionCommander
	scheduler *scheduler_dispatcher.Scheduler
	logger    *slog.Logger
}

// New создаёт новый экземпляр приложения
func New(b *telebot.Bot, svc interfaces.Service, subs interfaces.SubscriptionCommander, logger *slog.Logger, scheduler *scheduler_dispatcher.Scheduler) (*Bot, error) {
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
		go b.scheduler.Run(ctx)
	}
	go b.bot.Start()
}

// Stop останавливает бота
func (b *Bot) Stop() {
	b.bot.Stop()
}
