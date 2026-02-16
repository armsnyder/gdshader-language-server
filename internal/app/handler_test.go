// Copyright (c) 2026 Adam Snyder <https://armsnyder.com> and contributors
// SPDX-License-Identifier: MIT

package app_test

import (
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"

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
		name     string
		document string
		seek     func(document string) lsp.Position
		want     types.GomegaMatcher
	}{
		{
			name:     "Keyword",
			document: "shader_type spatial;\n",
			seek:     toMiddle("shader_type"),
			want:     valueTo(ContainSubstring("shader")),
		},
		{
			name:     "KeywordAtEnd",
			document: "shader_type spatial;\n",
			seek:     toAfter("shader_type"),
			want:     valueTo(ContainSubstring("Declares the shader type")),
		},
		{
			name:     "StencilModeKeyword",
			document: "shader_type spatial;\nstencil_mode compare_always;\n",
			seek:     toMiddle("stencil_mode"),
			want:     valueTo(ContainSubstring("Declares one or more stencil modes")),
		},
		{
			name:     "GroupUniformsAtEnd",
			document: "group_uniforms MyGroup;\n",
			seek:     toAfter("group_uniforms"),
			want:     valueTo(ContainSubstring("Groups uniforms together")),
		},
		{
			name:     "DataType",
			document: "uniform vec3 color;\n",
			seek:     toMiddle("vec3"),
			want:     valueTo(ContainSubstring("vector")),
		},
		{
			name:     "DataTypeIncludesGDScriptType",
			document: "uniform vec3 color;\n",
			seek:     toMiddle("vec3"),
			want:     valueTo(ContainSubstring("GDScript type:")),
		},
		{
			name:     "UniformHint",
			document: "uniform sampler2D tex : hint_normal;\n",
			seek:     toMiddle("hint_normal"),
			want:     valueTo(ContainSubstring("Used as normalmap")),
		},
		{
			name:     "UniformHintSourceColorVec",
			document: "uniform vec3 color : source_color;\n",
			seek:     toMiddle("source_color"),
			want:     valueTo(ContainSubstring("Used as color")),
		},
		{
			name:     "UniformHintSourceColorSampler",
			document: "uniform sampler2D tex : source_color;\n",
			seek:     toMiddle("source_color"),
			want:     valueTo(ContainSubstring("Used as albedo color")),
		},
		{
			name:     "UnknownWord",
			document: "uniform vec3 my_color;\n",
			seek:     toMiddle("my_color"),
			want:     BeNil(),
		},
		{
			name:     "Whitespace",
			document: "uniform   vec3 color;\n",
			seek:     toOffset("uniform", len("uniform")+2),
			want:     BeNil(),
		},
		{
			name:     "BuiltInConstant",
			document: "shader_type spatial;\nvoid vertex() {\nVERTEX = vec3(0.0);\n}\n",
			seek:     toMiddle("VERTEX"),
			want:     valueTo(ContainSubstring("Position of the vertex")),
		},
		{
			name:     "ShaderTypeSky",
			document: "shader_type sky;\n",
			seek:     toMiddle("sky"),
			want:     valueTo(ContainSubstring("Sky shader")),
		},
		{
			name:     "ShaderTypeParticles",
			document: "shader_type particles;\n",
			seek:     toMiddle("particles"),
			want:     valueTo(ContainSubstring("particle")),
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
					Position:     mustSeekPosition(t, tt.seek, tt.document),
				},
			})
			g.Expect(err).ToNot(HaveOccurred(), "Hover error")
			g.Expect(result).To(tt.want)
		})
	}
}

func valueTo(m types.GomegaMatcher) types.GomegaMatcher {
	return WithTransform(func(v any) any {
		if hover, ok := v.(*lsp.Hover); ok && hover != nil {
			return hover.Contents.Value
		}
		return nil
	}, m)
}
