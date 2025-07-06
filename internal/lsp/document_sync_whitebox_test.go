// Copyright (c) 2025 Adam Snyder <https://armsnyder.com> and contributors
// SPDX-License-Identifier: MIT

package lsp

import (
	"testing"

	. "github.com/onsi/gomega"
)

func deriveChange(before, after string) (change TextDocumentContentChangeEvent, startOffset, endOffset int) {
	// HACK: For simplicity, we assume all characters are ASCII.

	// Find the first difference
	for startOffset < len(before) && startOffset < len(after) && before[startOffset] == after[startOffset] {
		startOffset++
	}

	// Find the last difference
	endOffset = len(before)
	endAfter := len(after)
	for endOffset > startOffset && endAfter > startOffset && before[endOffset-1] == after[endAfter-1] {
		endOffset--
		endAfter--
	}

	getPosition := func(beforeIndex int) Position {
		var line, character int
		for _, b := range before[:beforeIndex] {
			if b == '\n' {
				line++
				character = 0
			} else {
				character++
			}
		}
		return Position{Line: line, Character: character}
	}

	return TextDocumentContentChangeEvent{
		Text:  after[startOffset:endAfter],
		Range: &Range{Start: getPosition(startOffset), End: getPosition(endOffset)},
	}, startOffset, endOffset
}

func TestUpdateLineStart(t *testing.T) {
	// Line offset tracking is tricky and deserves this whitebox test.

	tests := []struct {
		name   string
		before string
		after  string
	}{
		{
			name:   "insert at start of document",
			before: "\n",
			after:  "A\n",
		},
		{
			name:   "insert newline in middle of line",
			before: "hello world\n",
			after:  "hello\n world\n",
		},
		{
			name:   "delete across lines",
			before: "line1\nline2\nline3\n",
			after:  "li3\n",
		},
		{
			name:   "replace line with more newlines",
			before: "abc\ndef\n",
			after:  "abc\na\nb\nc\n\n",
		},
		{
			name:   "append to final line",
			before: "line1\n",
			after:  "line1\nline2\n",
		},
		{
			name:   "remove newline",
			before: "line1\nline2\nline3\n",
			after:  "line1line2\nline3\n",
		},
		{
			name:   "insert newline at end of file",
			before: "line1\nline2",
			after:  "line1\nline2\n",
		},
		{
			name:   "insert multiple newlines mid-line",
			before: "header: value\n",
			after:  "header:\nvalue\nextra\n",
		},
		{
			name:   "delete everything",
			before: "some\ntext\nhere\n",
			after:  "",
		},
		{
			name:   "no-op (identical text)",
			before: "no change\nhere\n",
			after:  "no change\nhere\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			lineStart := computeLineStart([]byte(tt.before))
			g.Expect(lineStart[0]).To(Equal(0), "lineStart[0] should be 0") // defensive
			change, startOffset, endOffset := deriveChange(tt.before, tt.after)
			t.Logf("before: %q", tt.before)
			t.Logf("after: %q", tt.after)
			t.Logf("change: %v", change)
			t.Logf("lineStart: %v", lineStart)
			t.Logf("startOffset: %d", startOffset)
			t.Logf("endOffset: %d", endOffset)
			got := updateLineStart(lineStart, change, startOffset, endOffset)
			want := computeLineStart([]byte(tt.after))
			g.Expect(want[0]).To(Equal(0), "want[0] should be 0") // defensive
			g.Expect(got).To(Equal(computeLineStart([]byte(tt.after))))
		})
	}
}
