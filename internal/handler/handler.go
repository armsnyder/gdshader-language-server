package handler

import (
	"log/slog"
	"slices"

	"github.com/armsnyder/gdshader-language-server/internal/lsp"
)

type Handler struct {
	isUTF8 bool
}

func (h *Handler) Initialize(clientCapabilities lsp.ClientCapabilities) (*lsp.ServerCapabilities, error) {
	var serverCapabilities lsp.ServerCapabilities
	serverCapabilities.TextDocumentSync.OpenClose = true

	h.isUTF8 = slices.Contains(clientCapabilities.General.PositionEncodings, "utf-8")

	if h.isUTF8 {
		serverCapabilities.PositionEncoding = "utf-8"
		serverCapabilities.TextDocumentSync.Change = lsp.SyncIncremental
	} else {
		slog.Warn("Client does not support utf-8 position encoding. Using full sync.")
		serverCapabilities.TextDocumentSync.Change = lsp.SyncFull
	}

	return &serverCapabilities, nil
}
