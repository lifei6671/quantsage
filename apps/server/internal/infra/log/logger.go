package log

import (
	"io"
	"log/slog"
	"os"

	"github.com/lifei6671/logit"
)

// New creates the default structured logger for QuantSage services.
func New() *slog.Logger {
	return newLogger(os.Stdout)
}

func newLogger(writer io.Writer) *slog.Logger {
	base := logit.NewSimpleLogger(writer, logit.WithMinLevel(logit.InfoLevel))
	return logit.NewSlogLogger(base, logit.WithSlogMinLevel(slog.LevelInfo))
}
