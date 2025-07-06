// Copyright (c) 2025 Adam Snyder <https://armsnyder.com> and contributors
// SPDX-License-Identifier: MIT

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
