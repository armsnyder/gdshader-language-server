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
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

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
	currentWord, c, err := h.getCompletionContext(params)
	if err != nil {
		return nil, fmt.Errorf("failed to get context: %w", err)
	}

	// TODO(asnyder):
	// - Swizzling https://docs.godotengine.org/en/stable/tutorials/shaders/shader_reference/shading_language.html#swizzling
	// - Constructors https://docs.godotengine.org/en/stable/tutorials/shaders/shader_reference/shading_language.html#constructing
	// - Array .length()
	// - Reference variables and functions
	// - Struct fields
	// - Built-in functions https://docs.godotengine.org/en/stable/tutorials/shaders/shader_reference/shader_functions.html#

	return &lsp.CompletionList{
		Items: lo.FilterMap(completionItems, func(item completionItemPredicate, _ int) (lsp.CompletionItem, bool) {
			return item.item, strings.HasPrefix(item.item.Label, currentWord) && item.predicate(*c)
		}),
	}, nil
}

func (h *Handler) getCompletionContext(params lsp.CompletionParams) (currentWord string, c *completionContext, err error) {
	doc, ok := h.Documents[params.TextDocument.URI]
	if !ok {
		return "", nil, fmt.Errorf("document not found: %s", params.TextDocument.URI)
	}

	lineStartPos := params.Position
	lineStartPos.Character = 0

	line, err := h.readBetweenPositions(doc, lineStartPos, params.Position)
	if err != nil {
		return "", nil, fmt.Errorf("reading current line: %w", err)
	}

	firstLine, err := h.readLine(doc, 0)
	if err != nil {
		return "", nil, fmt.Errorf("reading first line: %w", err)
	}

	c = &completionContext{}

	c.functionName, err = h.getCurrentFunction(doc, params.Position)
	if err != nil {
		return "", nil, fmt.Errorf("getting current function: %w", err)
	}

	firstLineTokens := tokenize(firstLine)
	if i := slices.Index(firstLineTokens, "shader_type"); i >= 0 && i < len(firstLineTokens)-1 {
		c.shaderType = firstLineTokens[i+1]
	}

	tokens := tokenize(line)
	if len(tokens) == 0 {
		return "", c, nil
	}

	c.lineTokens = tokens[:len(tokens)-1]
	return tokens[len(tokens)-1], c, nil
}

func (h *Handler) readBetweenPositions(doc *lsp.Document, startPos, endPos lsp.Position) ([]byte, error) {
	startOffset, err := doc.PositionToOffset(startPos)
	if err != nil {
		return nil, fmt.Errorf("start position to offset: %w", err)
	}

	var endOffset int
	if endPos.Line >= doc.Lines()-1 {
		endOffset = doc.Len()
	} else {
		endOffset, err = doc.PositionToOffset(endPos)
		if err != nil {
			return nil, fmt.Errorf("end position to offset: %w", err)
		}
	}

	return io.ReadAll(io.NewSectionReader(doc, int64(startOffset), int64(endOffset-startOffset)))
}

func (h *Handler) readLine(doc *lsp.Document, lineNumber int) ([]byte, error) {
	startPos := lsp.Position{Line: lineNumber, Character: 0}
	endPos := lsp.Position{Line: lineNumber + 1, Character: 0}

	line, err := h.readBetweenPositions(doc, startPos, endPos)
	if err != nil {
		return nil, fmt.Errorf("reading line %d: %w", lineNumber, err)
	}

	return line, nil
}

func (h *Handler) getCurrentFunction(doc *lsp.Document, pos lsp.Position) (string, error) {
	for lineNumber := pos.Line; lineNumber >= 0; lineNumber-- {
		line, err := h.readLine(doc, lineNumber)
		if err != nil {
			return "", fmt.Errorf("reading line %d: %w", lineNumber, err)
		}

		tokens := tokenize(line)
		for i := len(tokens) - 2; i > 1; i-- {
			if tokens[i] == ")" && tokens[i+1] == "{" {
				return tokens[1], nil
			}
		}
	}

	return "", nil
}

func tokenize(line []byte) []string {
	var tokens []string

	isNotWord := func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
	}

	for {
		// Skip over any whitespace.
		line = bytes.TrimLeftFunc(line, unicode.IsSpace)
		// Capture each punctuation as a separate token.
		if i := bytes.IndexFunc(line, isNotWord); i == 0 {
			r, size := utf8.DecodeRune(line)
			tokens = append(tokens, string(r))
			line = line[size:]
			continue
		}
		// Capture the word.
		if i := bytes.IndexFunc(line, isNotWord); i >= 0 {
			tokens = append(tokens, string(line[:i]))
			line = line[i:]
			continue
		}
		// This is the last token
		if len(line) > 0 {
			tokens = append(tokens, string(line))
		}
		return tokens
	}
}

var _ lsp.Handler = &Handler{}
