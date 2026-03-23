// Copyright (c) 2026 Adam Snyder <https://armsnyder.com> and contributors
// SPDX-License-Identifier: MIT

// Command gentables generates CSV files from the Godot shader reference
// documentation tables and function signatures. These tables drive completion,
// hover, and other data-driven language server features.
package main

import (
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func main() {
	gitSource := &GitHubFS{
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
		Repo: "godotengine/godot-docs",
		Ref:  "stable",
	}

	docsFS, err := fs.Sub(gitSource, "tutorials/shaders/shader_reference")
	if err != nil {
		slog.Error("Error creating sub FS", "error", err)
		os.Exit(1)
	}

	const outputRoot = "internal/app/tables"
	mixinFS := os.DirFS("internal/app/table_mixins")

	if err := GenerateTables(docsFS, mixinFS, outputRoot); err != nil {
		slog.Error("Error generating tables", "error", err)
		os.Exit(1)
	}

	slog.Info("Tables generated successfully", "output", outputRoot)
}
