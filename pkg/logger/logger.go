package logger

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/config"
)

// New создаёт и настраивает slog-логгер из уровня логирования
func New(cfg *config.LoggerConfig) *slog.Logger {
	var handler slog.Handler

	level, err := parseLevel(cfg.Level)
	if err != nil {
		level = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: replaceAttrs,
		AddSource:   true,
	}

	switch cfg.Format {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	case "", "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

// parseLevel преобразует строковый уровень в slog.Leveler.
func parseLevel(logLevel string) (slog.Leveler, error) {
	switch strings.ToLower(strings.TrimSpace(logLevel)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return nil, errors.New("unknown log level: " + logLevel)
	}
}

// levelString — аккуратное имя уровня.
func levelString(l slog.Level) string {
	switch {
	case l <= slog.LevelDebug:
		return "DEBUG"
	case l == slog.LevelInfo:
		return "INFO"
	case l == slog.LevelWarn:
		return "WARN"
	default:
		return "ERROR"
	}
}

func replaceAttrs(_ []string, a slog.Attr) slog.Attr {
	switch a.Key {
	case slog.TimeKey:
		// Безопасно приводим к time.Time (если формат подменён — оставим как есть)
		if tt, ok := a.Value.Any().(time.Time); ok {
			a.Value = slog.StringValue(tt.UTC().Format(time.RFC3339))
		} else {
			// fallback на прямой вызов (на случай стандартного времени)
			t := a.Value.Time()
			a.Value = slog.StringValue(t.UTC().Format(time.RFC3339))
		}
	case slog.LevelKey:
		// Уровень в ВЕРХНЕМ РЕГИСТРЕ, компактно
		if lv, ok := a.Value.Any().(slog.Level); ok {
			a.Value = slog.StringValue(strings.ToUpper(levelString(lv)))
		}
	case slog.SourceKey:
		// Сокращаем путь к файлу до base + :строка
		if src, ok := a.Value.Any().(*slog.Source); ok && src != nil {
			file := filepath.Base(src.File)
			a.Value = slog.StringValue(file + ":" + strconv.Itoa(src.Line))
		}
	}
	return a
}
