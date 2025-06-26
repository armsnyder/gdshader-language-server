package lsp

import "fmt"

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#clientCapabilities
type ClientCapabilities struct{}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#serverCapabilities
type ServerCapabilities struct {
	TextDocumentSync      *TextDocumentSyncOptions `json:"textDocumentSync,omitempty"`
	CompletionProvider    *CompletionOptions       `json:"completionProvider,omitempty"`
	HoverProvider         bool                     `json:"hoverProvider,omitempty"`
	DefinitionProvider    bool                     `json:"definitionProvider,omitempty"`
	ReferencesProvider    bool                     `json:"referencesProvider,omitempty"`
	SignatureHelpProvider bool                     `json:"signatureHelpProvider,omitempty"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocumentSyncOptions
type TextDocumentSyncOptions struct {
	OpenClose bool                 `json:"openClose,omitempty"`
	Change    TextDocumentSyncKind `json:"change"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#completionOptions
type CompletionOptions struct{}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocumentSyncKind
type TextDocumentSyncKind int

// Text document sync kinds.
const (
	SyncNone        TextDocumentSyncKind = 0
	SyncFull        TextDocumentSyncKind = 1
	SyncIncremental TextDocumentSyncKind = 2
)

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#initializeResult
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#responseError
type ResponseError struct {
	Code          ErrorCode `json:"code"`
	Message       string    `json:"message"`
	InternalError error     `json:"-"`
}

func (e *ResponseError) Error() string {
	msg := fmt.Sprintf("error code %d: %s", e.Code, e.Message)
	if e.InternalError != nil {
		msg += fmt.Sprintf(": %s", e.InternalError.Error())
	}
	return msg
}

func (e *ResponseError) Unwrap() error {
	return e.InternalError
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#errorCodes
type ErrorCode int

// LSP error codes.
const (
	CodeParseError     ErrorCode = -32700
	CodeInvalidRequest ErrorCode = -32600
	CodeMethodNotFound ErrorCode = -32601
	CodeInvalidParams  ErrorCode = -32602
	CodeInternalError  ErrorCode = -32603
)

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#didOpenTextDocumentParams
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#didChangeTextDocumentParams
type DidChangeTextDocumentParams struct {
	TextDocument   TextDocumentIdentifier           `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocumentContentChangeEvent
type TextDocumentContentChangeEvent struct {
	Text  string `json:"text"`
	Range *Range `json:"range,omitempty"`
}

func (e TextDocumentContentChangeEvent) String() string {
	return fmt.Sprintf("%q @ %s", e.Text, e.Range)
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#didCloseTextDocumentParams
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#position
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Character)
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#range
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

func (r Range) String() string {
	return fmt.Sprintf("%s-%s", r.Start, r.End)
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocumentItem
type TextDocumentItem struct {
	URI  string `json:"uri"`
	Text string `json:"text"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocumentIdentifier
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocumentPositionParams
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#location
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}
