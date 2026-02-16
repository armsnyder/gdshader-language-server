// Copyright (c) 2026 Adam Snyder <https://armsnyder.com> and contributors
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

func TestHandler_Hover(t *testing.T) {
	tests := []struct {
		name         string
		document     string
		position     lsp.Position
		wantNil      bool
		wantContains string
	}{
		{
			name:         "Keyword",
			document:     "shader_type spatial;\n",
			position:     lsp.Position{Line: 0, Character: 3},
			wantContains: "shader",
		},
		{
			name:         "DataType",
			document:     "uniform vec3 color;\n",
			position:     lsp.Position{Line: 0, Character: 9},
			wantContains: "vector",
		},
		{
			name:     "UnknownWord",
			document: "uniform vec3 my_color;\n",
			position: lsp.Position{Line: 0, Character: 15},
			wantNil:  true,
		},
		{
			name:     "Whitespace",
			document: "uniform   vec3 color;\n",
			position: lsp.Position{Line: 0, Character: 8},
			wantNil:  true,
		},
		{
			name:         "BuiltInConstant",
			document:     "shader_type spatial;\nvoid vertex() {\nVERTEX = vec3(0.0);\n}\n",
			position:     lsp.Position{Line: 2, Character: 2},
			wantContains: "vertex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			var h app.Handler
			const uri = "file:///test.gdshader"

			err := h.DidOpenTextDocument(t.Context(), lsp.DidOpenTextDocumentParams{
				TextDocument: lsp.TextDocumentItem{URI: uri, Text: tt.document},
			})
			g.Expect(err).ToNot(HaveOccurred(), "DidOpenTextDocument error")

			result, err := h.Hover(t.Context(), lsp.HoverParams{
				TextDocumentPositionParams: lsp.TextDocumentPositionParams{
					TextDocument: lsp.TextDocumentIdentifier{URI: uri},
					Position:     tt.position,
				},
			})
			g.Expect(err).ToNot(HaveOccurred(), "Hover error")

			if tt.wantNil {
				g.Expect(result).To(BeNil())
			} else {
				g.Expect(result).ToNot(BeNil())
				g.Expect(result.Contents.Value).To(ContainSubstring(tt.wantContains))
			}
		})
	}
}
