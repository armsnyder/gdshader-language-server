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

package main

import (
	_ "embed"
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/armsnyder/gdshader-language-server/internal/app"
	"github.com/armsnyder/gdshader-language-server/internal/lsp"
)

//go:embed version.txt
var version string

func main() {
	var flags struct {
		Debug bool
	}

	flag.BoolVar(&flags.Debug, "debug", false, "Enable debug logging")

	flag.Parse()

	setupLogger(flags.Debug)

	server := &lsp.Server{
		Info: lsp.ServerInfo{
			Name:    "gdshader-language-server",
			Version: strings.TrimSpace(version),
		},
		Handler: &app.Handler{},
	}

	if err := server.Serve(); err != nil {
		os.Exit(1)
	}
}

func setupLogger(debug bool) { //nolint:revive
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:     level,
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
