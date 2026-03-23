// Copyright (c) 2026 Adam Snyder <https://armsnyder.com> and contributors
// SPDX-License-Identifier: MIT

package app_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/armsnyder/gdshader-language-server/internal/lsp"
)

// toMiddle seeks to the middle of the first matching token.
func toMiddle(token string) func(document string) lsp.Position {
	return func(document string) lsp.Position {
		start := tokenStartIndexInDocument(document, token, 0)
		return positionForByteOffset(document, start+len(token)/2)
	}
}

// toAfter seeks to the end of the first matching token.
func toAfter(token string) func(document string) lsp.Position {
	return func(document string) lsp.Position {
		start := tokenStartIndexInDocument(document, token, 0)
		return positionForByteOffset(document, start+len(token))
	}
}

// toOffset seeks to the start of the first matching token plus delta bytes.
func toOffset(token string, delta int) func(document string) lsp.Position {
	return func(document string) lsp.Position {
		start := tokenStartIndexInDocument(document, token, 0)
		return positionForByteOffset(document, start+delta)
	}
}

// mustSeekPosition resolves a test-defined seek closure and converts panic to test failure.
func mustSeekPosition(t *testing.T, seek func(document string) lsp.Position, document string) (pos lsp.Position) {
	t.Helper()
	if seek == nil {
		t.Fatal("seek function must be set")
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("seek panic: %v", r)
		}
	}()
	return seek(document)
}

func tokenStartIndexInDocument(document, token string, occurrence int) int {
	if occurrence < 0 {
		panic("occurrence must be >= 0")
	}
	searchFrom := 0
	for i := 0; i <= occurrence; i++ {
		offset := strings.Index(document[searchFrom:], token)
		if offset < 0 {
			panic("token occurrence not found")
		}
		start := searchFrom + offset
		if i == occurrence {
			return start
		}
		searchFrom = start + len(token)
	}

	panic("unreachable")
}

func positionForByteOffset(document string, byteOffset int) lsp.Position {
	if byteOffset < 0 {
		byteOffset = 0
	}
	if byteOffset > len(document) {
		byteOffset = len(document)
	}

	prefix := document[:byteOffset]
	line := strings.Count(prefix, "\n")
	lastLineStart := strings.LastIndex(prefix, "\n") + 1
	char := utf8.RuneCountInString(prefix[lastLineStart:])

	return lsp.Position{Line: line, Character: char}
}
