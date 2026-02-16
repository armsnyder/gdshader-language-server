// Copyright (c) 2026 Adam Snyder <https://armsnyder.com> and contributors
// SPDX-License-Identifier: MIT

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
		HoverProvider:      true,
	}, nil
}

// Hover implements lsp.Handler.
func (h *Handler) Hover(_ context.Context, params lsp.HoverParams) (*lsp.Hover, error) {
	word, err := h.getWordAtPosition(params.TextDocumentPositionParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get word: %w", err)
	}

	if word == "" {
		return nil, nil
	}

	_, c, err := h.getCompletionContext(lsp.CompletionParams(params))
	if err != nil {
		return nil, fmt.Errorf("failed to get completion context: %w", err)
	}

	for _, item := range completionItems {
		if item.item.Label == word && item.item.Documentation != nil && item.predicate(*c) {
			return &lsp.Hover{
				Contents: *item.item.Documentation,
			}, nil
		}
	}

	return nil, nil
}

// getWordAtPosition returns the identifier under the cursor using tokenized line offsets.
func (h *Handler) getWordAtPosition(params lsp.TextDocumentPositionParams) (string, error) {
	doc, ok := h.Documents[params.TextDocument.URI]
	if !ok {
		return "", fmt.Errorf("document not found: %s", params.TextDocument.URI)
	}

	line, err := h.readLine(doc, params.Position.Line)
	if err != nil {
		return "", fmt.Errorf("reading line: %w", err)
	}

	tokens := tokenize(line)
	offset := 0
	for _, token := range tokens {
		// Search from the previous token end so repeated tokens map to the correct occurrence.
		tokenStart := strings.Index(string(line[offset:]), token)
		if tokenStart < 0 {
			continue
		}
		tokenStart += offset
		tokenEnd := tokenStart + len(token)
		if isWithinToken(params.Position.Character, tokenStart, tokenEnd, token) {
			r, _ := utf8.DecodeRuneInString(token)
			if unicode.IsLetter(r) || r == '_' {
				return token, nil
			}
			return "", nil
		}
		offset = tokenEnd
	}

	return "", nil
}

// isWithinToken handles both in-token and end-of-token cursor positions for hover compatibility.
func isWithinToken(charPos, tokenStart, tokenEnd int, token string) bool {
	if charPos >= tokenStart && charPos < tokenEnd {
		return true
	}
	// Some clients report hover positions at the end of a word.
	if charPos == tokenEnd {
		r, _ := utf8.DecodeRuneInString(token)
		return unicode.IsLetter(r) || r == '_'
	}
	return false
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

// getCompletionContext derives shader type, current function, and prior tokens for predicate filtering.
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

// readBetweenPositions reads raw bytes between two LSP positions in one document.
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

// readLine reads one line by translating line boundaries into document byte ranges.
func (h *Handler) readLine(doc *lsp.Document, lineNumber int) ([]byte, error) {
	startPos := lsp.Position{Line: lineNumber, Character: 0}
	endPos := lsp.Position{Line: lineNumber + 1, Character: 0}

	line, err := h.readBetweenPositions(doc, startPos, endPos)
	if err != nil {
		return nil, fmt.Errorf("reading line %d: %w", lineNumber, err)
	}

	return line, nil
}

// getCurrentFunction returns the innermost function containing the cursor position.
func (h *Handler) getCurrentFunction(doc *lsp.Document, pos lsp.Position) (string, error) {
	text, err := h.readBetweenPositions(doc, lsp.Position{Line: 0, Character: 0}, pos)
	if err != nil {
		return "", fmt.Errorf("reading document prefix: %w", err)
	}

	return currentFunctionFromPrefix(text), nil
}

// currentFunctionFromPrefix scans backwards through source bytes and finds the nearest enclosing function.
func currentFunctionFromPrefix(prefix []byte) string {
	depth := 0
	for i := len(prefix) - 1; i >= 0; i-- {
		if prefix[i] == '}' {
			depth++
			continue
		}
		if prefix[i] != '{' {
			continue
		}
		if depth > 0 {
			depth--
			continue
		}
		if name := functionNameBeforeBrace(prefix[:i]); name != "" {
			return name
		}
	}

	return ""
}

// functionNameBeforeBrace extracts a likely function name from the nearest header before an opening brace.
func functionNameBeforeBrace(prefix []byte) string {
	header := string(prefix)
	if lineStart := strings.LastIndexByte(header, '\n'); lineStart >= 0 {
		header = header[lineStart+1:]
	}
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}

	beforeParen, _, hasParen := strings.Cut(header, "(")
	if !hasParen {
		return ""
	}
	parts := strings.Fields(beforeParen)
	if len(parts) < 2 {
		return ""
	}

	return parts[len(parts)-1]
}

// tokenize splits a line into word tokens and standalone punctuation tokens.
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
