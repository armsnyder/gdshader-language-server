package lsp

import "fmt"

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#clientCapabilities
type ClientCapabilities struct {
	General struct {
		PositionEncodings []string `json:"positionEncodings,omitempty"`
	} `json:"general"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#serverCapabilities
type ServerCapabilities struct {
	PositionEncoding   string                  `json:"positionEncoding,omitempty"`
	TextDocumentSync   TextDocumentSyncOptions `json:"textDocumentSync"`
	CompletionProvider bool                    `json:"completionProvider,omitempty"`
	HoverProvider      bool                    `json:"hoverProvider,omitempty"`
	DefinitionProvider bool                    `json:"definitionProvider,omitempty"`
	ReferencesProvider bool                    `json:"referencesProvider,omitempty"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocumentSyncOptions
type TextDocumentSyncOptions struct {
	OpenClose bool                 `json:"openClose,omitempty"`
	Change    TextDocumentSyncKind `json:"change"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocumentSyncKind
type TextDocumentSyncKind int

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
	internalError error     `json:"-"`
}

func (e *ResponseError) Error() string {
	msg := fmt.Sprintf("error code %d: %s", e.Code, e.Message)
	if e.internalError != nil {
		msg += fmt.Sprintf(": %s", e.internalError.Error())
	}
	return msg
}

func (e *ResponseError) Unwrap() error {
	return e.internalError
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#errorCodes
type ErrorCode int

const (
	CodeParseError     ErrorCode = -32700
	CodeInvalidRequest ErrorCode = -32600
	CodeMethodNotFound ErrorCode = -32601
	CodeInvalidParams  ErrorCode = -32602
	CodeInternalError  ErrorCode = -32603
)
