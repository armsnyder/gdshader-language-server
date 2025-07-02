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
