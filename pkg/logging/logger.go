package logging

import (
	"log/slog"
	"os"
	"sync"
)

var (
	logger *slog.Logger
	once   sync.Once
)

func InitLogger() {
	once.Do(func() {
		handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
			ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
				if attr.Key == slog.TimeKey {
					attr.Value = slog.StringValue(attr.Value.Time().Format("2006-01-02T15:04:05"))
				}
				return attr
			},
		})
		logger = slog.New(handler)
	})
}

func GetLogger() *slog.Logger {
	if logger == nil {
		InitLogger()
	}
	return logger
}

func StringField(key, value string) slog.Attr {
	return slog.String(key, value)
}

func ErrorField(err error) slog.Attr {
	if err == nil {
		return slog.String("error", "")
	}
	return slog.String("error", err.Error())
}
