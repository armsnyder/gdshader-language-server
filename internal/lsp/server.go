// Copyright (c) 2025 Adam Snyder <https://armsnyder.com> and contributors
// SPDX-License-Identifier: MIT

package lsp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/textproto"
	"os"
	"strconv"
)

// DocumentSyncHandler defines methods for handling document synchronization.
type DocumentSyncHandler interface {
	DidOpenTextDocument(ctx context.Context, params DidOpenTextDocumentParams) error
	DidCloseTextDocument(ctx context.Context, params DidCloseTextDocumentParams) error
	DidChangeTextDocument(ctx context.Context, params DidChangeTextDocumentParams) error
}

// Handler provides the logic for handling LSP requests and notifications.
type Handler interface {
	DocumentSyncHandler
	Initialize(ctx context.Context, clientCapabilities ClientCapabilities) (*ServerCapabilities, error)
	Completion(ctx context.Context, params CompletionParams) (*CompletionList, error)
}

// Server manages the LSP server lifecycle and dispatching requests and
// notifications to a handler.
type Server struct {
	Stdin   io.Reader
	Stdout  io.Writer
	Info    ServerInfo
	Handler Handler
}

// Serve runs the LSP server. It blocks until the client receives an "exit".
func (s *Server) Serve() error {
	if s.Stdin == nil {
		s.Stdin = os.Stdin
	}

	scanner := bufio.NewScanner(s.Stdin)
	scanner.Split(jsonRPCSplit)

	slog.Info("Server is running", "name", s.Info.Name, "version", s.Info.Version)

	for scanner.Scan() {
		if !s.processMessage(scanner.Bytes()) {
			return nil
		}
	}

	slog.Error("Scanner error", "error", scanner.Err())

	return scanner.Err()
}

func (s *Server) processMessage(payload []byte) bool {
	var request RequestMessage
	if err := json.Unmarshal(payload, &request); err != nil {
		slog.Error("Bad request", "error", err)
		return true
	}

	if len(request.ID) == 0 {
		logger := slog.With("method", request.Method)
		logger.Debug("Received notification", "params", string(request.Params))

		if request.Method == "exit" {
			logger.Info("Exiting")
			return false
		}

		if err := s.handleNotification(request.Method, request.Params); err != nil {
			logger.Error("Error handling notification", "error", err)
		}

		return true
	}

	logger := slog.With("request_id", request.ID, "method", request.Method)
	debugEnabled := logger.Enabled(context.TODO(), slog.LevelDebug)

	if debugEnabled {
		logger.Debug("Received request", "params", string(request.Params))
	}

	result, err := s.handleRequest(request.Method, request.Params)
	if err == nil {
		err = s.writeMessage(&ResponseMessage{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result:  result,
		})
	} else {
		logger.Error("Error handling request", "error", err)
		var asResponseError *ResponseError
		if !errors.As(err, &asResponseError) {
			asResponseError = &ResponseError{
				Code:    CodeInternalError,
				Message: err.Error(),
			}
		}
		err = s.writeMessage(&ErrorResponseMessage{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error:   asResponseError,
		})
	}

	if err != nil {
		logger.Error("Failed to write response", "error", err)
	}

	if debugEnabled {
		logger.Debug("Sent response", "response", fmt.Sprintf("%#v", result))
	}

	return true
}

func jsonRPCSplit(data []byte, _ bool) (advance int, token []byte, err error) {
	const headerDelimiter = "\r\n\r\n"

	i := bytes.Index(data, []byte(headerDelimiter))
	if i == -1 {
		return 0, nil, nil
	}

	payloadIndex := i + len(headerDelimiter)

	header, err := textproto.NewReader(bufio.NewReader(bytes.NewReader(data[:payloadIndex]))).ReadMIMEHeader()
	if err != nil {
		return 0, nil, fmt.Errorf("bad header: %w", err)
	}

	contentLength, err := strconv.Atoi(header.Get("content-length"))
	if err != nil {
		return 0, nil, fmt.Errorf("bad content-length: %w", err)
	}

	restBytes := data[payloadIndex:]

	if len(restBytes) < contentLength {
		return 0, nil, nil
	}

	return payloadIndex + contentLength, restBytes[:contentLength], nil
}

func (s *Server) handleNotification(method string, paramsRaw json.RawMessage) error {
	switch method {
	case "initialized":

	case "$/cancelRequest":
		// TODO(asnyder): Handle cancelRequest and make everything
		// async.

	case "textDocument/didOpen":
		var params DidOpenTextDocumentParams
		if err := parseParams(paramsRaw, &params); err != nil {
			return err
		}
		return s.Handler.DidOpenTextDocument(context.TODO(), params)

	case "textDocument/didClose":
		var params DidCloseTextDocumentParams
		if err := parseParams(paramsRaw, &params); err != nil {
			return err
		}
		return s.Handler.DidCloseTextDocument(context.TODO(), params)

	case "textDocument/didChange":
		var params DidChangeTextDocumentParams
		if err := parseParams(paramsRaw, &params); err != nil {
			return err
		}
		return s.Handler.DidChangeTextDocument(context.TODO(), params)

	default:
		slog.Warn("Unknown notification", "method", method)
	}
	return nil
}

func (s *Server) handleRequest(method string, paramsRaw json.RawMessage) (any, error) {
	switch method {
	case "initialize":
		// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#initializeParams
		var params struct {
			ClientInfo struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"clientInfo"`
			Capabilities ClientCapabilities `json:"capabilities"`
		}
		if err := parseParams(paramsRaw, &params); err != nil {
			return nil, err
		}

		slog.Info("Client info", "name", params.ClientInfo.Name, "version", params.ClientInfo.Version)

		serverCapabilities, err := s.Handler.Initialize(context.TODO(), params.Capabilities)
		if err != nil {
			return nil, err
		}

		// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#initializeResult
		return struct {
			Capabilities *ServerCapabilities `json:"capabilities"`
			ServerInfo   ServerInfo          `json:"serverInfo"`
		}{Capabilities: serverCapabilities, ServerInfo: s.Info}, nil

	case "shutdown":
		return nil, nil

	case "textDocument/completion":
		var params CompletionParams
		if err := parseParams(paramsRaw, &params); err != nil {
			return nil, err
		}
		return s.Handler.Completion(context.TODO(), params)

	default:
		return nil, &ResponseError{
			Code:    CodeMethodNotFound,
			Message: fmt.Sprintf("Unknown method %q", method),
		}
	}
}

func parseParams(paramsRaw json.RawMessage, result any) error {
	if err := json.Unmarshal(paramsRaw, result); err != nil {
		return &ResponseError{
			Code:          CodeInvalidParams,
			Message:       err.Error(),
			InternalError: err,
		}
	}
	return nil
}

func (s *Server) writeMessage(message Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}

	if err := s.writeRaw(data); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	return nil
}

func (s *Server) writeRaw(data []byte) error {
	if s.Stdout == nil {
		s.Stdout = os.Stdout
	}

	_, err := s.Stdout.Write(append([]byte("Content-Length: "+strconv.Itoa(len(data))+"\r\nContent-Type: application/vscode-jsonrpc; charset=utf-8\r\n\r\n"), data...))
	return err
}
