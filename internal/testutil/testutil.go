// MIT License
//
// Copyright (c) 2025 Adam Snyder
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

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
