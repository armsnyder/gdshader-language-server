package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/armsnyder/gdshader-language-server/internal/handler"
	"github.com/armsnyder/gdshader-language-server/internal/lsp"
)

var Version = "development"

func main() {
	var flags struct {
		Debug bool
	}

	flag.BoolVar(&flags.Debug, "debug", false, "Enable debug logging")

	flag.Parse()

	if flags.Debug {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		})))
	}

	server := &lsp.Server{
		Info: lsp.ServerInfo{
			Name:    "gdshader-language-server",
			Version: Version,
		},
		Handler: &handler.Handler{},
	}

	if err := server.Serve(); err != nil {
		os.Exit(1)
	}
}
