package xlog

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Config configures a zerolog-backed slog handler.
type Config struct {
	// Level is the minimum enabled slog level. Defaults to INFO.
	Level slog.Level
	// Format controls output layout: "json" (default) or "console".
	Format string
	// Output sets the destination writer. Defaults to stdout.
	Output io.Writer
	// TimeFormat controls timestamps in output. Defaults to RFC3339Nano.
	TimeFormat string
}

// NewLogger returns a slog.Logger backed by zerolog.
func NewLogger(configs ...Config) *slog.Logger {
	return slog.New(NewHandler(configs...))
}

// NewHandler returns a slog.Handler backed by zerolog.
func NewHandler(configs ...Config) slog.Handler {
	cfg := defaultConfig()
	if len(configs) > 0 {
		provided := configs[0]
		if provided.Level != 0 {
			cfg.Level = provided.Level
		}
		if provided.Format != "" {
			cfg.Format = strings.ToLower(provided.Format)
		}
		if provided.Output != nil {
			cfg.Output = provided.Output
		}
		if provided.TimeFormat != "" {
			cfg.TimeFormat = provided.TimeFormat
		}
	}

	zerolog.TimeFieldFormat = cfg.TimeFormat

	writer := cfg.Output
	if cfg.Format == "console" {
		writer = zerolog.ConsoleWriter{Out: cfg.Output, TimeFormat: cfg.TimeFormat}
	}

	zl := zerolog.New(writer).Level(toZeroLevel(cfg.Level)).With().Timestamp().Logger()

	return &handler{logger: zl, level: cfg.Level}
}

type handler struct {
	logger zerolog.Logger
	level  slog.Level
	attrs  []slog.Attr
	groups []string
}

func (h *handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *handler) Handle(_ context.Context, record slog.Record) error {
	event := h.logger.WithLevel(toZeroLevel(record.Level))
	if event == nil {
		return nil
	}

	for _, attr := range h.attrs {
		appendAttr(event, attr, h.groups)
	}

	record.Attrs(func(attr slog.Attr) bool {
		appendAttr(event, attr, h.groups)
		return true
	})

	event.Msg(record.Message)
	return nil
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := h.clone()
	next.attrs = append(next.attrs, attrs...)
	return next
}

func (h *handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	next := h.clone()
	next.groups = append(next.groups, name)
	return next
}

func (h *handler) clone() *handler {
	attrs := make([]slog.Attr, len(h.attrs))
	copy(attrs, h.attrs)

	groups := make([]string, len(h.groups))
	copy(groups, h.groups)

	return &handler{
		logger: h.logger,
		level:  h.level,
		attrs:  attrs,
		groups: groups,
	}
}

func appendAttr(event *zerolog.Event, attr slog.Attr, groups []string) {
	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) {
		return
	}

	key := buildKey(attr.Key, groups)

	switch attr.Value.Kind() {
	case slog.KindBool:
		event.Bool(key, attr.Value.Bool())
	case slog.KindDuration:
		event.Dur(key, attr.Value.Duration())
	case slog.KindFloat64:
		event.Float64(key, attr.Value.Float64())
	case slog.KindInt64:
		event.Int64(key, attr.Value.Int64())
	case slog.KindString:
		event.Str(key, attr.Value.String())
	case slog.KindTime:
		event.Time(key, attr.Value.Time())
	case slog.KindUint64:
		event.Uint64(key, attr.Value.Uint64())
	case slog.KindGroup:
		for _, groupAttr := range attr.Value.Group() {
			appendAttr(event, groupAttr, append(groups, attr.Key))
		}
	case slog.KindAny:
		event.Interface(key, attr.Value.Any())
	default:
		event.Interface(key, attr.Value.Any())
	}
}

func buildKey(key string, groups []string) string {
	if len(groups) == 0 {
		return key
	}
	parts := make([]string, 0, len(groups)+1)
	parts = append(parts, groups...)
	parts = append(parts, key)
	return strings.Join(parts, ".")
}

func defaultConfig() Config {
	return Config{
		Level:      slog.LevelInfo,
		Format:     "json",
		Output:     os.Stdout,
		TimeFormat: time.RFC3339Nano,
	}
}

func toZeroLevel(level slog.Level) zerolog.Level {
	switch {
	case level <= slog.LevelDebug:
		return zerolog.DebugLevel
	case level < slog.LevelWarn:
		return zerolog.InfoLevel
	case level < slog.LevelError:
		return zerolog.WarnLevel
	default:
		return zerolog.ErrorLevel
	}
}

// AttrsFromMap converts a map to sorted slog attrs for stable logs and tests.
func AttrsFromMap(fields map[string]interface{}) []slog.Attr {
	if len(fields) == 0 {
		return nil
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	attrs := make([]slog.Attr, 0, len(keys))
	for _, key := range keys {
		attrs = append(attrs, slog.Any(key, fields[key]))
	}
	return attrs
}
