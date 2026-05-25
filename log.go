package hatchcert

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"sync"
)

const (
	LogSilent       = 1
	LogOnlyImporant = 2
	LogVerbose      = 3
)

// Logger provides buffering capabilities for slog messages and a mechanism for
// always-visible output in specific modes.
type Logger struct {
	mu                  sync.Mutex
	buf                 bytes.Buffer
	alwaysOutputHandler slog.Handler
}

// Important prints a message directly to os.Stdout if the logger is configured
// for direct output (e.g., in timer mode). Otherwise, it logs it as an Info
// message using the default logger.
func (l *Logger) Important(msg string, args ...any) {
	if l.alwaysOutputHandler != nil {
		slog.New(l.alwaysOutputHandler).Info(msg, args...)
	} else {
		slog.Info(msg, args...)
	}
}

// Write implements the io.Writer interface, directing output to the internal buffer.
func (l *Logger) Write(p []byte) (n int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}

// Emit flushes any buffered log messages to os.Stdout and then clears the buffer.
func (l *Logger) Emit() {
	if l != nil {
		l.mu.Lock()
		defer l.mu.Unlock()
		os.Stdout.Write(l.buf.Bytes())
		l.buf.Reset()
	}
}

// SetupLogger initializes the default slog.Logger and returns a Logger instance.
// Adjusts to the requested verbosity level. With default options, does not
// output anything unless a failure occurs when running under cron.
func SetupLogger(level int) *Logger {
	var handler slog.Handler
	l := &Logger{}

	if level == LogVerbose {
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		var target io.Writer = os.Stderr
		if level == LogSilent {
			target = l
		}
		handler = slog.NewTextHandler(target, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
		if level != LogSilent {
			l.alwaysOutputHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			})
		}
	}

	slog.SetDefault(slog.New(handler))
	return l
}
