package hatchcert

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"sync"
)

// Logger provides buffering capabilities for slog messages. Instances can
// safely be nil.
type Logger struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

// Write implements the io.Writer interface, directing output to the internal buffer.
func (l *Logger) Write(p []byte) (n int, err error) {
	if l != nil {
		l.mu.Lock()
		defer l.mu.Unlock()
		return l.buf.Write(p)
	}
	return len(p), nil
}

// Emit flushes any buffered log messages to os.Stdout and then clears the buffer.
func (l *Logger) Emit() {
	if l != nil {
		l.mu.Lock()
		defer l.mu.Unlock()
		if b := l.buf.Bytes(); len(b) > 0 {
			fmt.Fprintln(os.Stderr, "The following additional logs were captured:")
			os.Stderr.Write(b)
		}
		l.buf.Reset()
	}
}

type LogOutputs struct {
	OnError *Logger
	ACME    *slog.Logger
}

func VerboseLogger() *slog.Logger {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	sl := slog.New(handler)
	slog.SetDefault(sl)
	return sl
}

// SetupLogger initializes the default slog.Logger. Unless in verbose mode,
// sets up log capturing for the ACME client. ACME logs will only be presented
// when a failure occurs.
func SetupLogger(verbose bool) (out LogOutputs) {
	if verbose {
		handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
		sl := slog.New(handler)
		slog.SetDefault(sl)
		out.OnError = nil
		out.ACME = sl
		return
	}

	var handler slog.Handler
	l := &Logger{}

	captureAcme := slog.NewTextHandler(l, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	out.OnError = l
	out.ACME = slog.New(captureAcme)

	handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
	return
}
