package log

import (
	"log/slog"
	"os"
)

// New creates the default structured logger for QuantSage services.
func New() *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	return slog.New(handler)
}
