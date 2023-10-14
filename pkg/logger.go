package pkg

import (
	"log/slog"
	"os"
)

func NewLogger(level slog.Level) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level}
	return slog.New(slog.NewJSONHandler(os.Stderr, opts))
}
