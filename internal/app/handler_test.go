// Copyright (c) 2025 Adam Snyder <https://armsnyder.com> and contributors
// SPDX-License-Identifier: MIT

package app_test

import (
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	. "github.com/onsi/gomega"

	"github.com/armsnyder/gdshader-language-server/internal/app"
	"github.com/armsnyder/gdshader-language-server/internal/lsp"
)

func TestHandler(t *testing.T) {
	t.Run("completion in empty document", func(t *testing.T) {
		g := NewWithT(t)
		var h app.Handler
		const uri = "file:///test.gdshader"

		// Open the document.
		err := h.DidOpenTextDocument(t.Context(), lsp.DidOpenTextDocumentParams{
			TextDocument: lsp.TextDocumentItem{URI: uri},
		})
		g.Expect(err).ToNot(HaveOccurred(), "DidOpenTextDocument error")

		// Type the first character.
		err = h.DidChangeTextDocument(t.Context(), lsp.DidChangeTextDocumentParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: uri},
			ContentChanges: []lsp.TextDocumentContentChangeEvent{{
				Range: &lsp.Range{},
				Text:  "s",
			}},
		})
		g.Expect(err).ToNot(HaveOccurred(), "DidChangeTextDocument error")

		// Get autocompletion list.
		list, err := h.Completion(t.Context(), lsp.CompletionParams{
			TextDocumentPositionParams: lsp.TextDocumentPositionParams{
				TextDocument: lsp.TextDocumentIdentifier{URI: uri},
				Position:     lsp.Position{Character: 1},
			},
		})
		g.Expect(err).ToNot(HaveOccurred(), "Completion error")
		expectedItem := lsp.CompletionItem{Label: "shader_type", Kind: lsp.CompletionKeyword}
		ignoreFields := cmpopts.IgnoreFields(lsp.CompletionItem{}, "Detail", "Documentation")
		g.Expect(list.Items).To(ContainElement(BeComparableTo(expectedItem, ignoreFields)), "Missing expected completion item")
	})
}
