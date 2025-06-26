package lsp_test

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/armsnyder/gdshader-language-server/internal/lsp"
	"github.com/armsnyder/gdshader-language-server/internal/testutil"
)

var bufferImplementations = []lsp.Buffer{
	&lsp.ArrayBuffer{},
	&lsp.GapBuffer{},
	&lsp.RopeBuffer{},
}

func newBuffer(buf lsp.Buffer) lsp.Buffer {
	return reflect.New(reflect.TypeOf(buf).Elem()).Interface().(lsp.Buffer) //nolint:revive
}

func bufferName(impl lsp.Buffer) string {
	return strings.Split(fmt.Sprintf("%T", impl), ".")[1]
}

func makeChanges(args ...string) []lsp.TextDocumentContentChangeEvent {
	var changes []lsp.TextDocumentContentChangeEvent
	for i := 0; i < len(args); i += 2 {
		changes = append(changes, makeChange(args[i], args[i+1]))
	}
	return changes
}

func makeChange(text, rangeStr string) lsp.TextDocumentContentChangeEvent {
	if rangeStr == "" {
		return lsp.TextDocumentContentChangeEvent{Text: text}
	}
	// Parse range string: "0:1-2:2"
	parts := strings.Split(rangeStr, "-")
	if len(parts) != 2 {
		panic(fmt.Sprintf("invalid range string: %q", rangeStr))
	}
	start := parsePos(parts[0])
	end := parsePos(parts[1])
	return lsp.TextDocumentContentChangeEvent{
		Text:  text,
		Range: &lsp.Range{Start: start, End: end},
	}
}

func parsePos(s string) lsp.Position {
	fields := strings.Split(s, ":")
	if len(fields) != 2 {
		panic(fmt.Sprintf("invalid position string: %q", s))
	}
	line, err1 := strconv.Atoi(fields[0])
	char, err2 := strconv.Atoi(fields[1])
	if err1 != nil || err2 != nil {
		panic(fmt.Sprintf("invalid position numbers in: %q", s))
	}
	return lsp.Position{Line: line, Character: char}
}

func TestDocument_ApplyChange(t *testing.T) {
	tests := []struct {
		name    string
		initial string
		changes []lsp.TextDocumentContentChangeEvent
		want    string
	}{
		{
			name:    "full reset",
			initial: "hello",
			changes: makeChanges("world", ""),
			want:    "world",
		},
		{
			name:    "insert at start of document",
			initial: "world",
			changes: makeChanges("hello ", "0:0-0:0"),
			want:    "hello world",
		},
		{
			name:    "insert at end of document",
			initial: "hello",
			changes: makeChanges(" world", "0:5-0:5"),
			want:    "hello world",
		},
		{
			name:    "insert newline in middle",
			initial: "hello world",
			changes: makeChanges("\n", "0:5-0:5"),
			want:    "hello\n world",
		},
		{
			name:    "delete single character",
			initial: "hello",
			changes: makeChanges("", "0:1-0:2"),
			want:    "hllo",
		},
		{
			name:    "delete across lines",
			initial: "line1\nline2\nline3",
			changes: makeChanges("", "0:4-2:4"),
			want:    "line3",
		},
		{
			name:    "replace single char",
			initial: "hello",
			changes: makeChanges("a", "0:1-0:2"),
			want:    "hallo",
		},
		{
			name:    "replace across lines",
			initial: "abc\ndef\nghi",
			changes: makeChanges("Z", "0:1-2:2"),
			want:    "aZi",
		},
		{
			name:    "insert emoji (surrogate pair)",
			initial: "hello",
			changes: makeChanges(" 👋", "0:5-0:5"),
			want:    "hello 👋",
		},
		{
			name:    "delete emoji",
			initial: "hi 👋 there",
			changes: makeChanges("", "0:3-0:5"),
			want:    "hi  there",
		},
		{
			name:    "repeated inserts ahead of newline",
			initial: "\n",
			changes: makeChanges("A", "0:0-0:0", "B", "0:1-0:1", "C", "0:2-0:2"),
			want:    "ABC\n",
		},
	}

	for _, impl := range bufferImplementations {
		t.Run(bufferName(impl), func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					g := NewWithT(t)
					testutil.SetupLogger(t)
					doc := lsp.NewDocument([]byte(tt.initial), newBuffer(impl))
					for i, change := range tt.changes {
						err := doc.ApplyChange(change)
						g.Expect(err).ToNot(HaveOccurred(), "ApplyChange #%d failed", i+1)
					}
					g.Expect(string(doc.Bytes())).To(BeComparableTo(tt.want))
				})
			}
		})
	}
}

func TestDocument_ApplyChange_Error(t *testing.T) {
	tests := []struct {
		name    string
		initial string
		change  lsp.TextDocumentContentChangeEvent
	}{
		{
			name:    "out of bounds line",
			initial: "hi",
			change:  makeChange("oops", "1:0-1:1"),
		},
		{
			name:    "character out of bounds in line",
			initial: "hi",
			change:  makeChange("oops", "0:99-0:99"),
		},
	}

	for _, impl := range bufferImplementations {
		t.Run(bufferName(impl), func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					doc := lsp.NewDocument([]byte(tt.initial), newBuffer(impl))
					NewWithT(t).Expect(doc.ApplyChange(tt.change)).ToNot(Succeed())
				})
			}
		})
	}
}

func TestDocument_ApplyChange_Allocs(t *testing.T) {
	doc := lsp.NewDocument([]byte("hello world"), &lsp.ArrayBuffer{})
	change := lsp.TextDocumentContentChangeEvent{Text: "new text", Range: &lsp.Range{}}
	allocs := testing.AllocsPerRun(10, func() {
		if err := doc.ApplyChange(change); err != nil {
			t.Fatal(err)
		}
	})
	if allocs > 0 {
		t.Errorf("ApplyChange allocated %f times, expected 0", allocs)
	}
}

func TestGapBuffer(t *testing.T) {
	t.Run("Delete does not malloc", func(t *testing.T) {
		g := NewWithT(t)
		var buf lsp.GapBuffer
		allocs := testing.AllocsPerRun(10, func() {
			buf.Reset([]byte("hello world"))
			for i := 10; i >= 0; i-- {
				buf.Delete(i, i)
			}
		})
		g.Expect(allocs).To(BeEquivalentTo(1))
	})

	t.Run("WriteAt does not malloc", func(t *testing.T) {
		g := NewWithT(t)
		var buf lsp.GapBuffer
		buf.Reset([]byte("hello world"))
		allocs := testing.AllocsPerRun(10, func() {
			for i := range 10 {
				_, _ = buf.WriteAt([]byte("x"), int64(i))
			}
		})
		g.Expect(allocs).To(BeEquivalentTo(0))
	})
}

func BenchmarkDocument(b *testing.B) {
	funcs := map[string]func(b *testing.B, impl lsp.Buffer){
		"Typing":      benchmarkTyping,
		"RandomEdits": benchmarkRandomEdits,
		"General":     benchmarkGeneral,
	}

	for name, fn := range funcs {
		b.Run(name, func(b *testing.B) {
			for _, impl := range bufferImplementations {
				b.Run(bufferName(impl), func(b *testing.B) {
					fn(b, impl)
				})
			}
		})
	}
}

func benchmarkTyping(b *testing.B, impl lsp.Buffer) {
	const initialLineCount = 100_000
	const initialLineLength = 100
	const editCount = 1000

	initial := bytes.Repeat(append(bytes.Repeat([]byte("x"), initialLineLength), '\n'), initialLineCount)

	for b.Loop() {
		doc := lsp.NewDocument(initial, newBuffer(impl))

		for i := range editCount {
			pos := lsp.Position{
				Line:      initialLineCount / 2,
				Character: initialLineLength/2 + i,
			}
			change := lsp.TextDocumentContentChangeEvent{
				Range: &lsp.Range{Start: pos, End: pos},
				Text:  "a",
			}

			if err := doc.ApplyChange(change); err != nil {
				b.Fatalf("ApplyChange failed at i=%d: %v", i, err)
			}
		}

		// Simulate a re-parse
		_ = doc.Bytes()
	}
}

func benchmarkRandomEdits(b *testing.B, impl lsp.Buffer) {
	const initialLineCount = 100_000
	const initialLineLength = 100
	const editCount = 1000

	var sb bytes.Buffer
	for range initialLineCount {
		sb.Write(bytes.Repeat([]byte("x"), initialLineLength))
		sb.WriteByte('\n')
	}
	initial := sb.String()

	rng := rand.New(rand.NewSource(42)) // deterministic randomness

	for b.Loop() {
		doc := lsp.NewDocument([]byte(initial), newBuffer(impl))

		for i := range editCount {
			pos := lsp.Position{
				Line:      rng.Intn(initialLineCount),
				Character: rng.Intn(initialLineLength),
			}
			change := lsp.TextDocumentContentChangeEvent{
				Range: &lsp.Range{Start: pos, End: pos},
				Text:  "abc",
			}

			if err := doc.ApplyChange(change); err != nil {
				b.Fatalf("ApplyChange failed at i=%d: %v", i, err)
			}
		}

		// Simulate a re-parse
		_ = doc.Bytes()
	}
}

func benchmarkGeneral(b *testing.B, impl lsp.Buffer) {
	const initialLineCount = 10_000

	var sb strings.Builder
	var lastLineLength int
	for i := range initialLineCount {
		line := fmt.Sprintf("this is line %d of the document\n", i)
		lastLineLength = len(line)
		sb.WriteString(line)
	}
	initial := sb.String()

	bursts := [][]lsp.TextDocumentContentChangeEvent{
		// Simulate typing "foo" at line 100
		{
			makeChange("f", "100:10-100:10"),
			makeChange("o", "100:11-100:11"),
			makeChange("o", "100:12-100:12"),
		},
		// Simulate typing "bar" at line 5000
		{
			makeChange("b", "5000:20-5000:20"),
			makeChange("a", "5000:21-5000:21"),
			makeChange("r", "5000:22-5000:22"),
		},
		// Undo: delete all of line 5000
		{makeChange("", "5000:0-5001:0")},
		// Simulate replacing part of a word at line 8000
		{makeChange("updated", "8000:5-8000:10")},
		// Simulate inserting a new line at the end
		{makeChange("new line added\n", fmt.Sprintf("%[1]d:%[2]d-%[1]d:%[2]d", initialLineCount-2, lastLineLength))},
	}

	for b.Loop() {
		doc := lsp.NewDocument([]byte(initial), newBuffer(impl))

		for i, burst := range bursts {
			for j, change := range burst {
				if err := doc.ApplyChange(change); err != nil {
					b.Fatalf("ApplyChange failed: burst=%d change=%d: %v", i, j, err)
				}
			}

			// Simulate a re-parse after each burst
			_ = doc.Bytes()
		}
	}
}
