package config

import (
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Загрузка конфигурации из config.yaml через cleanenv

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Scheduler SchedulerConfig `yaml:"scheduler"`
	Postgres  PostgresConfig  `yaml:"postgres"`
	CoinGecko CoinGeckoConfig `yaml:"coingecko"`
	Telegram  TelegramConfig  `yaml:"telegram"`
	Logger    LoggerConfig    `yaml:"logger"`
}

type ServerConfig struct {
	Addr            string        `yaml:"addr" env-default:":8080"`
	ReadTimeout     time.Duration `yaml:"read_timeout" env-default:"5s"`
	WriteTimeout    time.Duration `yaml:"write_timeout" env-default:"10s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" env-default:"60s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env-default:"10s"`
}

type SchedulerConfig struct {
	Enabled  bool          `yaml:"enabled" env-default:"true"`
	Interval time.Duration `yaml:"interval" env-default:"5m"`
}

type LoggerConfig struct {
	Level  string `yaml:"level"  env-default:"info"` // debug|info|warn|error
	Format string `yaml:"format" env-default:"text"` // text|json
}

type PostgresConfig struct {
	Host            string        `yaml:"host" env-default:"localhost"`
	Port            int           `yaml:"port" env-default:"5432"`
	User            string        `yaml:"user" env-default:"postgres"`
	Password        string        `yaml:"password" env-default:"postgres"`
	DBName          string        `yaml:"dbname" env-default:"crypto"`
	SSLMode         string        `yaml:"sslmode" env-default:"disable"`
	Timeout         time.Duration `yaml:"timeout" env-default:"5s"`
	MaxConns        int32         `yaml:"max_conns" env-default:"10"`
	MinConns        int32         `yaml:"min_conns" env-default:"1"`
	MaxConnLifetime time.Duration `yaml:"max_conn_lifetime" env-default:"1h"`
	MaxConnIdleTime time.Duration `yaml:"max_conn_idle_time" env-default:"30m"`
}

type CoinGeckoConfig struct {
	BaseURL   string        `yaml:"base_url"`
	Coins     []string      `yaml:"coins"`
	Currency  string        `yaml:"currency"`
	Timeout   time.Duration `yaml:"timeout" env-default:"8s"`
	UserAgent string        `yaml:"user_agent" env-default:"crypto-rate-service/1.0"`
}

type TelegramConfig struct {
	Enabled             bool   `yaml:"enabled" env-default:"false"`
	Token               string `yaml:"token" env:"TELEGRAM_BOT_TOKEN" env-required:"true"`
	DefaultAutoInterval int    `yaml:"default_auto_interval" env-default:"10"` // minutes
}

func LoadConfig(path string) (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
