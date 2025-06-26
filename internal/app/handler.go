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

package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/armsnyder/gdshader-language-server/internal/lsp"
	"github.com/samber/lo"
)

// Handler encapsulates the logic of the Godot shader language server.
type Handler struct {
	lsp.Filesystem
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

// Completion implements lsp.Handler.
func (h *Handler) Completion(_ context.Context, params lsp.CompletionParams) (*lsp.CompletionList, error) {
	currentWord, err := h.getCurrentWord(params)
	if err != nil {
		return nil, fmt.Errorf("failed to get current word: %w", err)
	}

	// TODO(asnyder):
	// - Swizzling https://docs.godotengine.org/en/stable/tutorials/shaders/shader_reference/shading_language.html#swizzling
	// - Constructors https://docs.godotengine.org/en/stable/tutorials/shaders/shader_reference/shading_language.html#constructing
	// - Array .length()
	// - Reference variables and functions
	// - Struct fields
	// - Built-in functions https://docs.godotengine.org/en/stable/tutorials/shaders/shader_reference/shader_functions.html#

	return &lsp.CompletionList{
		Items: lo.Filter(completionItems, func(item lsp.CompletionItem, _ int) bool {
			return strings.HasPrefix(item.Label, currentWord)
		}),
	}, nil
}

func (h *Handler) getCurrentWord(params lsp.CompletionParams) (string, error) {
	doc, ok := h.Documents[params.TextDocument.URI]
	if !ok {
		return "", fmt.Errorf("document not found: %s", params.TextDocument.URI)
	}

	lineStartPos := params.Position
	lineStartPos.Character = 0
	lineStartOffset, err := doc.PositionToOffset(lineStartPos)
	if err != nil {
		return "", fmt.Errorf("get line offset: %w", err)
	}

	currentOffset, err := doc.PositionToOffset(params.Position)
	if err != nil {
		return "", fmt.Errorf("get current offset: %w", err)
	}

	line, err := io.ReadAll(io.NewSectionReader(doc, int64(lineStartOffset), int64(currentOffset-lineStartOffset)))
	if err != nil {
		return "", fmt.Errorf("reading line: %w", err)
	}

	i := bytes.LastIndexFunc(line, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	})
	if i == -1 {
		return string(line), nil
	}
	return string(line[i+1:]), nil
}

var _ lsp.Handler = &Handler{}
