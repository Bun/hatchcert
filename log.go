package hatchcert

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"sync"
)

type Logger struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *Logger) Write(p []byte) (n int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}

func (l *Logger) Emit() {
	if l != nil {
		l.mu.Lock()
		defer l.mu.Unlock()
		os.Stdout.Write(l.buf.Bytes())
		l.buf.Reset()
	}
}

func SetupLogger(verbose bool) *Logger {
	var w io.Writer
	var l *Logger
	var opts *slog.HandlerOptions
	if verbose {
		w = os.Stderr
		opts = &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}
	} else {
		// If not verbose, we only output on error
		l = &Logger{}
		w = l
	}
	logger := slog.New(slog.NewTextHandler(w, opts))
	slog.SetDefault(logger)
	return l
}
