package bot

import (
	"context"
	"log/slog"
	"time"

	"gopkg.in/telebot.v4"
)

// Config — конфигурация бота
type Config struct {
	Token           string
	LongPollTimeout time.Duration
}

// RateDTO — данные о курсе валюты
type RateDTO struct {
	Symbol    string
	Price     float64
	Min24h    float64
	Max24h    float64
	Change1h  float64
	UpdatedAt time.Time
}

// RatesReader — интерфейс для чтения курсов валют
type RatesReader interface {
	GetCurrencyRates(ctx context.Context) ([]RateDTO, error)
	GetCurrencyRateBySymbol(ctx context.Context, symbol string) (RateDTO, error)
}

// SubscriptionStore — интерфейс для управления подписками
type SubscriptionStore interface {
	Enable(ctx context.Context, chatID int64, intervalMinutes int) error
	Disable(ctx context.Context, chatID int64) error
	Due(ctx context.Context, now time.Time) ([]int64, error)
	MarkSent(ctx context.Context, chatID int64, at time.Time) error
}

// Bot — основной тип приложения
type Bot struct {
	bot       *telebot.Bot
	rates     RatesReader
	subs      SubscriptionStore
	scheduler *scheduler
	logger    *slog.Logger
}

// New создаёт новый экземпляр приложения
func New(cfg Config, rates RatesReader, subs SubscriptionStore, logger *slog.Logger) (*Bot, error) {
	if cfg.LongPollTimeout <= 0 {
		cfg.LongPollTimeout = 10 * time.Second
	}

	b, err := telebot.NewBot(telebot.Settings{
		Token:  cfg.Token,
		Poller: &telebot.LongPoller{Timeout: cfg.LongPollTimeout},
	})
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		bot:    b,
		rates:  rates,
		subs:   subs,
		logger: logger,
	}

	// маршруты команд
	b.Handle("/start", bot.handleStart)
	b.Handle("/rates", bot.handleRates)
	b.Handle("/startauto", bot.handleStartAuto)
	b.Handle("/stopauto", bot.handleStopAuto)
	bot.scheduler = newScheduler(b, rates, subs, time.Minute, logger)
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
