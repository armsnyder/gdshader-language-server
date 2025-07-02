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

package lsp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/zyedidia/rope"
)

// BufferType represents a buffer implementation.
type BufferType int

const (
	// BufferTypeDefault chooses a buffer for you.
	BufferTypeDefault BufferType = iota
	// BufferTypeGap uses a gap buffer for amortized O(1) insertions and
	// deletions and fast reads, at the cost of poorer random access.
	BufferTypeGap
	// BufferTypeRope uses a rope data structure for efficient random
	// access and insertions/deletions in large documents.
	BufferTypeRope
)

// Filesystem can be embedded into handlers in order to implement the basic
// document sync methods of the LSP.
type Filesystem struct {
	Documents  map[string]*Document
	BufferType BufferType
}

// DidOpenTextDocument implements DocumentSyncHandler.
func (f *Filesystem) DidOpenTextDocument(_ context.Context, params DidOpenTextDocumentParams) error {
	var buf Buffer
	switch f.BufferType {
	case BufferTypeGap:
		buf = &GapBuffer{}
	case BufferTypeRope:
		buf = &RopeBuffer{}
	}

	if f.Documents == nil {
		f.Documents = make(map[string]*Document)
	}
	f.Documents[params.TextDocument.URI] = NewDocument([]byte(params.TextDocument.Text), buf)

	return nil
}

// DidCloseTextDocument implements lsp.Handler.
func (f *Filesystem) DidCloseTextDocument(_ context.Context, params DidCloseTextDocumentParams) error {
	delete(f.Documents, params.TextDocument.URI)
	return nil
}

// DidChangeTextDocument implements lsp.Handler.
func (f *Filesystem) DidChangeTextDocument(_ context.Context, params DidChangeTextDocumentParams) error {
	doc, ok := f.Documents[params.TextDocument.URI]
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

var _ DocumentSyncHandler = (*Filesystem)(nil)

// Buffer implements large text storage with methods for random access.
type Buffer interface {
	io.ReaderAt
	io.WriterAt
	// Reset reinitializes the buffer with the given text.
	Reset(b []byte)
	// Bytes returns the full content of the buffer as a byte slice.
	Bytes() []byte
	// Delete deletes a range of bytes from the buffer.
	Delete(start, end int)
	// Len returns the number of bytes in the buffer.
	Len() int
}

// Document represents a text document with methods to manipulate its content.
type Document struct {
	buffer    Buffer
	lineStart []int
	cache     []byte
	charBuf   []byte
}

// NewDocument creates a new Document with the given initial text and buffer.
// If buf is nil, a GapBuffer is used.
func NewDocument(text []byte, buf Buffer) *Document {
	if buf == nil {
		buf = &GapBuffer{}
	}
	doc := &Document{buffer: buf, charBuf: make([]byte, 1024)}
	doc.Reset(text)
	return doc
}

// Reset reinitializes the document with the given text.
func (d *Document) Reset(text []byte) {
	d.cache = nil
	d.buffer.Reset(text)
	d.lineStart = computeLineStart(text)
}

// Bytes returns the full content of the document.
func (d *Document) Bytes() []byte {
	if d.cache != nil {
		return d.cache
	}
	d.cache = d.buffer.Bytes()
	return d.cache
}

// ReadAt implements io.ReaderAt.
func (d *Document) ReadAt(p []byte, off int64) (n int, err error) {
	if d.cache != nil {
		return copy(p, d.cache[off:]), nil
	}
	return d.buffer.ReadAt(p, off)
}

// Len returns the number of bytes in the document.
func (d *Document) Len() int {
	if d.cache != nil {
		return len(d.cache)
	}
	return d.buffer.Len()
}

// ApplyChange applies a content change to the document.
func (d *Document) ApplyChange(change TextDocumentContentChangeEvent) error {
	d.cache = nil

	if len(d.charBuf) == 0 {
		d.charBuf = make([]byte, 1024)
	}

	if change.Range == nil {
		d.Reset([]byte(change.Text))
		return nil
	}

	startOffset, endOffset, err := d.getChangeOffsets(change)
	if err != nil {
		return fmt.Errorf("get change offsets: %w", err)
	}

	if startOffset != endOffset {
		d.buffer.Delete(startOffset, endOffset)
	}

	if change.Text != "" {
		if err := d.writeText(change.Text, startOffset); err != nil {
			return fmt.Errorf("write text at offset %d: %w", startOffset, err)
		}
	}

	d.lineStart = updateLineStart(d.lineStart, change, startOffset, endOffset)

	return nil
}

func (d *Document) getChangeOffsets(change TextDocumentContentChangeEvent) (start, end int, err error) {
	startOffset, err := d.PositionToOffset(change.Range.Start)
	if err != nil {
		return 0, 0, err
	}

	// Optimize for basic typing, where end == start
	endOffset := startOffset
	if change.Range.End != change.Range.Start {
		endOffset, err = d.PositionToOffset(change.Range.End)
		if err != nil {
			return 0, 0, err
		}
	}

	return startOffset, endOffset, nil
}

func (d *Document) writeText(text string, off int) error {
	var toWrite []byte
	if len([]byte(text)) <= len(d.charBuf) {
		n := copy(d.charBuf, text)
		toWrite = d.charBuf[:n]
	} else {
		toWrite = []byte(text)
	}
	_, err := d.buffer.WriteAt(toWrite, int64(off))
	return err
}

// PositionToOffset converts a Position (line and character) to a byte offset
// in the document. It correctly handles UTF-16 character widths.
func (d *Document) PositionToOffset(pos Position) (int, error) {
	if pos.Line >= len(d.lineStart) {
		return 0, fmt.Errorf("invalid line: %d", pos.Line)
	}

	start, end := d.lineBounds(pos.Line)

	offset, u16Count := start, 0
	for offset < end {
		chunkSize := min(len(d.charBuf), end-offset)

		n, err := d.buffer.ReadAt(d.charBuf[:chunkSize], int64(offset))
		if err != nil && !errors.Is(err, io.EOF) {
			return 0, fmt.Errorf("buffer read at line %d: %w", pos.Line, err)
		}

		if n == 0 {
			break
		}

		deltaOffset, done, err := decodeUntilTargetOffset(d.charBuf[:n], pos.Character, &u16Count)
		if err != nil {
			return 0, err
		}
		if done {
			return offset + deltaOffset, nil
		}

		offset += n
	}

	if u16Count >= pos.Character {
		return offset, nil
	}

	return 0, fmt.Errorf("line %d: target units %d out of bounds (only %d utf16 units)", pos.Line, pos.Character, u16Count)
}

func (d *Document) lineBounds(line int) (start, end int) {
	start = d.lineStart[line]
	if line+1 < len(d.lineStart) {
		return start, d.lineStart[line+1]
	}
	return start, d.buffer.Len()
}

func decodeUntilTargetOffset(buf []byte, targetU16Offset int, u16Count *int) (deltaOffset int, done bool, err error) {
	for i := 0; i < len(buf); {
		r, size := utf8.DecodeRune(buf[i:])
		if r == utf8.RuneError && size == 1 {
			return 0, false, fmt.Errorf("invalid utf-8 at byte offset %d", i)
		}
		if *u16Count >= targetU16Offset {
			return i, true, nil
		}
		*u16Count += utf16Width(r)
		i += size
	}
	return 0, false, nil
}

func utf16Width(r rune) int {
	if r <= 0xFFFF {
		return 1
	}
	return 2
}

// Lines returns the number of lines in the document. A single line ending in
// a newline character is counted as two lines. This is consistent with the
// LSP specification.
func (d *Document) Lines() int {
	return len(d.lineStart)
}

// ArrayBuffer is the simplest implementation of Buffer, using a byte slice
// for storage. It is optimized for reads. Insertions and deletions are O(n)
// due to slice copying.
//
// This implementation is not recommended and is mainly used as a testing
// benchmark baseline for smarter buffer implementations.
type ArrayBuffer struct {
	data []byte
}

// Bytes implements Buffer.
func (a *ArrayBuffer) Bytes() []byte {
	return a.data
}

// Delete implements Buffer.
func (a *ArrayBuffer) Delete(start, end int) {
	a.data = slices.Delete(a.data, start, end)
}

// Len implements Buffer.
func (a *ArrayBuffer) Len() int {
	return len(a.data)
}

// ReadAt implements Buffer.
func (a *ArrayBuffer) ReadAt(p []byte, off int64) (n int, err error) {
	n = copy(p, a.data[off:])
	return n, nil
}

// Reset implements Buffer.
func (a *ArrayBuffer) Reset(b []byte) {
	a.data = b
}

// WriteAt implements Buffer.
func (a *ArrayBuffer) WriteAt(p []byte, off int64) (n int, err error) {
	a.data = slices.Insert(a.data, int(off), p...)
	return len(p), nil
}

var _ Buffer = (*ArrayBuffer)(nil)

// GapBuffer implements a gap buffer for amortized O(1) insertions and
// deletions at the cursor position and O(n) for random access. Its reads
// are fast compared to [RopeBuffer].
type GapBuffer struct {
	buf      []byte
	gapStart int
	gapEnd   int
}

const initialGapSize = 128

// Bytes implements Buffer.
func (g *GapBuffer) Bytes() []byte {
	return append(g.buf[:g.gapStart], g.buf[g.gapEnd:]...)
}

// Delete implements Buffer.
func (g *GapBuffer) Delete(start, end int) {
	count := end - start
	switch g.gapStart {
	// Specific cases where we do not need to copy data
	case start:
		g.gapEnd += count
	case end:
		g.gapStart -= count
	// General case where we put the buffer into a state optimized for a
	// follow-up write to the start offset.
	default:
		g.moveGapTo(start)
		g.gapEnd += count
	}
}

// Len implements Buffer.
func (g *GapBuffer) Len() int {
	return len(g.buf) - g.gapSize()
}

// ReadAt implements Buffer.
func (g *GapBuffer) ReadAt(p []byte, off int64) (n int, err error) {
	if int(off) < g.gapStart {
		n = copy(p, g.buf[int(off):g.gapStart])
		n += copy(p[n:], g.buf[g.gapEnd:])
	} else {
		n = copy(p, g.buf[g.physicalOffset(int(off)):])
	}
	if n < len(p) {
		err = io.EOF
	}
	return n, err
}

// Reset implements Buffer.
func (g *GapBuffer) Reset(b []byte) {
	g.buf = make([]byte, len(b)+initialGapSize) // preallocate a gap
	copy(g.buf, b)
	g.gapStart = len(b)
	g.gapEnd = len(g.buf)
}

// WriteAt implements Buffer.
func (g *GapBuffer) WriteAt(p []byte, off int64) (n int, err error) {
	g.moveGapTo(int(off))
	g.growGap(len(p) - g.gapSize())
	n = copy(g.buf[g.gapStart:], p)
	g.gapStart += n
	g.shrinkGapTo(1024)
	return n, nil
}

func (g *GapBuffer) physicalOffset(off int) int {
	if off < g.gapStart {
		return off
	}
	return off + g.gapSize()
}

func (g *GapBuffer) gapSize() int {
	return g.gapEnd - g.gapStart
}

func (g *GapBuffer) moveGapTo(off int) {
	count := off - g.gapStart
	switch {
	case count < 0:
		copy(g.buf[g.gapEnd+count:], g.buf[off:g.gapStart])
	case count > 0:
		copy(g.buf[g.gapStart:g.gapStart+count], g.buf[g.gapEnd:g.gapEnd+count])
	}
	g.gapEnd += count
	g.gapStart += count
}

func (g *GapBuffer) shrinkGapTo(n int) {
	excess := g.gapSize() - n
	if excess <= 0 {
		return
	}
	g.buf = slices.Delete(g.buf, g.gapStart, g.gapStart+excess)
	g.gapEnd -= excess
}

func (g *GapBuffer) growGap(n int) {
	if n <= 0 {
		return
	}
	n += initialGapSize
	g.buf = slices.Insert(g.buf, g.gapEnd, make([]byte, n)...)
	g.gapEnd += n
}

var _ Buffer = (*GapBuffer)(nil)

// RopeBuffer implements Buffer using a rope data structure. This is best at
// scale, for large documents with frequent random insertions and deletions.
type RopeBuffer struct {
	node *rope.Node
}

// Bytes implements Buffer.
func (r *RopeBuffer) Bytes() []byte {
	return r.node.Value()
}

// Delete implements Buffer.
func (r *RopeBuffer) Delete(start, end int) {
	r.node.Remove(start, end)
}

// Len implements Buffer.
func (r *RopeBuffer) Len() int {
	return r.node.Len()
}

// ReadAt implements Buffer.
func (r *RopeBuffer) ReadAt(p []byte, off int64) (n int, err error) {
	return r.node.ReadAt(p, off)
}

// Reset implements Buffer.
func (r *RopeBuffer) Reset(b []byte) {
	r.node = rope.New(b)
}

// WriteAt implements Buffer.
func (r *RopeBuffer) WriteAt(p []byte, off int64) (n int, err error) {
	r.node.Insert(int(off), p)
	return len(p), nil
}

var _ Buffer = (*RopeBuffer)(nil)

func computeLineStart(text []byte) []int {
	lineStart := []int{0}
	for i, b := range text {
		if b == '\n' {
			lineStart = append(lineStart, i+1)
		}
	}
	return lineStart
}

func updateLineStart(lineStart []int, change TextDocumentContentChangeEvent, startOffset, endOffset int) []int {
	// 1. We know all offsets before the change will remain the same.
	// 2. Since we know the text inserted, we can calculate any new line
	//    offsets that are created by the change.
	// 3. Since we know the length of the text inserted, and we are passed
	//    the length of the text removed, we can use those to update the
	//    offsets of the lines after the change.

	newLinesInserted := strings.Count(change.Text, "\n")
	newLinesRemoved := change.Range.End.Line - change.Range.Start.Line
	growth := newLinesInserted - newLinesRemoved

	switch {
	case growth > 0:
		// Shift the existing elements right
		lineStart = slices.Insert(lineStart, change.Range.End.Line+1, make([]int, growth)...)
	case growth < 0:
		// Shift the existing elements left
		lineStart = slices.Delete(lineStart, change.Range.End.Line+growth+1, change.Range.End.Line+1)
	}

	if newLinesInserted > 0 {
		// Save the new offsets
		asBytes := []byte(change.Text)
		off := 0
		for i := range newLinesInserted {
			off += bytes.IndexByte(asBytes[off:], '\n') + 1
			lineStart[change.Range.Start.Line+i+1] = startOffset + off
		}
	}

	// Update the offsets for lines after the change
	for i := change.Range.End.Line + 1 + growth; i < len(lineStart); i++ {
		lineStart[i] += len([]byte(change.Text)) + startOffset - endOffset
	}

	return lineStart
}
