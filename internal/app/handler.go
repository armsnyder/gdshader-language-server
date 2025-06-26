package app

import (
	"context"
	"fmt"

	"github.com/armsnyder/gdshader-language-server/internal/lsp"
)

// Handler encapsulates the logic of the Godot shader language server.
type Handler struct {
	documents map[string]*lsp.Document
}

// Initialize implements lsp.Handler.
func (h *Handler) Initialize(context.Context, lsp.ClientCapabilities) (*lsp.ServerCapabilities, error) {
	return &lsp.ServerCapabilities{
		TextDocumentSync: &lsp.TextDocumentSyncOptions{
			OpenClose: true,
			Change:    lsp.SyncIncremental,
		},
		CompletionProvider: &lsp.CompletionOptions{},
	}, nil
}

// DidOpenTextDocument implements lsp.Handler.
func (h *Handler) DidOpenTextDocument(_ context.Context, params lsp.DidOpenTextDocumentParams) error {
	doc := lsp.NewDocument([]byte(params.TextDocument.Text), nil)
	if h.documents == nil {
		h.documents = make(map[string]*lsp.Document)
	}
	h.documents[params.TextDocument.URI] = doc
	return nil
}

// DidCloseTextDocument implements lsp.Handler.
func (h *Handler) DidCloseTextDocument(_ context.Context, params lsp.DidCloseTextDocumentParams) error {
	delete(h.documents, params.TextDocument.URI)
	return nil
}

// DidChangeTextDocument implements lsp.Handler.
func (h *Handler) DidChangeTextDocument(_ context.Context, params lsp.DidChangeTextDocumentParams) error {
	doc, ok := h.documents[params.TextDocument.URI]
	if !ok {
		return fmt.Errorf("document not found: %s", params.TextDocument.URI)
	}

	for _, change := range params.ContentChanges {
		if err := doc.ApplyChange(change); err != nil {
			return err
		}
	}

	return nil
}

var _ lsp.Handler = &Handler{}
