package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)

type Options struct {
	Level     string // debug, info, warn, error
	Format    Format // json, text
	AddSource bool
	Output    io.Writer // defaults to os.Stdout
}

var defaultLogger = slog.New(slog.NewTextHandler(os.Stdout, nil))

// Init builds a logger from Options, sets it as the process default, and
// returns it. Call it once at startup before any other package logs.
func Init(opts Options) *slog.Logger {
	output := opts.Output
	if output == nil {
		output = os.Stdout
	}

	handlerOpts := &slog.HandlerOptions{
		Level:     ParseLevel(opts.Level),
		AddSource: opts.AddSource,
	}

	var handler slog.Handler
	if opts.Format == FormatJSON {
		handler = slog.NewJSONHandler(output, handlerOpts)
	} else {
		handler = slog.NewTextHandler(output, handlerOpts)
	}

	l := slog.New(handler)
	slog.SetDefault(l)
	defaultLogger = l
	return l
}

// ParseLevel maps a level name to a slog.Level, defaulting to Info.
func ParseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// L returns the current process-wide logger.
func L() *slog.Logger {
	return defaultLogger
}

func Debug(msg string, args ...any) { defaultLogger.Debug(msg, args...) }
func Info(msg string, args ...any)  { defaultLogger.Info(msg, args...) }
func Warn(msg string, args ...any)  { defaultLogger.Warn(msg, args...) }
func Error(msg string, args ...any) { defaultLogger.Error(msg, args...) }

// Fatal logs at error level then terminates the process, mirroring
// log.Fatalf so it's a drop-in replacement.
func Fatal(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
	os.Exit(1)
}

func DebugContext(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).DebugContext(ctx, msg, args...)
}

func InfoContext(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).InfoContext(ctx, msg, args...)
}

func WarnContext(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).WarnContext(ctx, msg, args...)
}

func ErrorContext(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).ErrorContext(ctx, msg, args...)
}

// With returns a logger that always includes the given key-value pairs,
// e.g. logger.With("component", "kafka-consumer").
func With(args ...any) *slog.Logger {
	return defaultLogger.With(args...)
}

type ctxKey struct{}

// WithContext attaches a logger to ctx so it can be retrieved later via
// FromContext, e.g. to carry request-scoped fields like a request ID.
func WithContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext returns the logger attached to ctx, or the process default
// logger if none was attached.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return l
	}
	return defaultLogger
}
