package testutil

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"
)

// SetupLogger configures the default slog logger for testing.
func SetupLogger(t testing.TB) {
	originalHandler := slog.Default().Handler()

	t.Cleanup(func() {
		slog.SetDefault(slog.New(originalHandler))
	})

	slog.SetDefault(slog.New(slog.NewTextHandler(TestWriter{t}, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey:
				// Suppress time
				return slog.Attr{}
			case slog.SourceKey:
				// Simplify file name
				source := a.Value.Any().(*slog.Source) //nolint:revive
				source.File = filepath.Base(source.File)
			}
			return a
		},
	})))
}

// TestWriter is an io.Writer that writes to test logs.
type TestWriter struct {
	T testing.TB
}

// Write implements io.Writer.
func (t TestWriter) Write(p []byte) (n int, err error) {
	t.T.Helper()
	t.T.Logf("%s", p)
	return len(p), nil
}

var _ io.Writer = TestWriter{}
