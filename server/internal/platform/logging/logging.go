// Package logging builds the logger used across the server (OPS-06).
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
)

const (
	// FormatJSON is for production: one JSON object per line for log collectors.
	FormatJSON = "json"
	// FormatText is easier to read in a terminal during development.
	FormatText = "text"
)

func New(w io.Writer, level, format string) (*slog.Logger, error) {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("PM_LOG_LEVEL: %w", err)
	}
	opts := &slog.HandlerOptions{Level: lvl}
	switch format {
	case FormatJSON:
		return slog.New(slog.NewJSONHandler(w, opts)), nil
	case FormatText:
		return slog.New(slog.NewTextHandler(w, opts)), nil
	default:
		return nil, fmt.Errorf("PM_LOG_FORMAT must be %s or %s, got %q", FormatJSON, FormatText, format)
	}
}

// AtLeast returns a logger that drops records below minLevel, so a chatty library
// stays quiet even when the application itself logs at debug level.
func AtLeast(log *slog.Logger, minLevel slog.Level) *slog.Logger {
	return slog.New(minLevelHandler{next: log.Handler(), min: minLevel})
}

type minLevelHandler struct {
	next slog.Handler
	min  slog.Level
}

func (h minLevelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.min && h.next.Enabled(ctx, level)
}

func (h minLevelHandler) Handle(ctx context.Context, r slog.Record) error {
	return h.next.Handle(ctx, r)
}

func (h minLevelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return minLevelHandler{next: h.next.WithAttrs(attrs), min: h.min}
}

func (h minLevelHandler) WithGroup(name string) slog.Handler {
	return minLevelHandler{next: h.next.WithGroup(name), min: h.min}
}
