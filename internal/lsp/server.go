package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/textproto"
	"os"
	"strconv"
)

type Handler interface {
	Initialize(clientCapabilities ClientCapabilities) (*ServerCapabilities, error)
}

type Server struct {
	Stdin   io.Reader
	Stdout  io.Writer
	Info    ServerInfo
	Handler Handler
}

func (s *Server) Serve() error {
	if s.Stdin == nil {
		s.Stdin = os.Stdin
	}

	scanner := bufio.NewScanner(s.Stdin)
	scanner.Split(jsonRPCSplit)

	slog.Info("Server is running", "name", s.Info.Name, "version", s.Info.Version)

	for scanner.Scan() {
		// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#requestMessage
		var request struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}

		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
			slog.Error("Bad request", "error", err)
			continue
		}

		if len(request.ID) == 0 {
			logger := slog.With("method", request.Method)
			logger.Debug("Received notification", "params", string(request.Params))

			if request.Method == "exit" {
				logger.Info("Exiting")
				return nil
			}

			if err := s.handleNotification(request.Method, request.Params); err != nil {
				logger.Error("Error handling notification", "error", err)
			}

			continue
		}

		logger := slog.With("request_id", request.ID, "method", request.Method)
		logger.Debug("Received request", "params", string(request.Params))

		response, err := s.handleRequest(request.Method, request.Params)
		if err != nil {
			logger.Error("Error handling request", "error", err)
			var asResponseError *ResponseError
			if errors.As(err, &asResponseError) {
				response = asResponseError
			} else {
				response = &ResponseError{
					Code:    CodeInternalError,
					Message: err.Error(),
				}
			}
		}

		if err := s.write(request.ID, response); err != nil {
			logger.Error("Write error", "error", err)
			continue
		}

		logger.Debug("Sent response")
	}

	slog.Error("Scanner error", "error", scanner.Err())

	return scanner.Err()
}

func jsonRPCSplit(data []byte, atEOF bool) (advance int, token []byte, err error) {
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
	case "cancelRequest":
		// TODO(asnyder): Handle cancelRequest
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

		serverCapabilities, err := s.Handler.Initialize(params.Capabilities)
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
			internalError: err,
		}
	}
	return nil
}

func (s *Server) write(requestID json.RawMessage, result any) error {
	// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#responseMessage
	message := struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Result  any             `json:"result"`
	}{JSONRPC: "2.0", ID: requestID, Result: result}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("invalid response: %w", err)
	}

	if s.Stdout == nil {
		s.Stdout = os.Stdout
	}

	_, err = s.Stdout.Write(append([]byte("Content-Length: "+strconv.Itoa(len(data))+"\r\nContent-Type: application/vscode-jsonrpc; charset=utf-8\r\n\r\n"), data...))
	return err
}
