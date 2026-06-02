// Package lsp implements a stateful, persistent JSON-RPC Language Server Protocol (LSP) client
// for multiplexing language analysis queries directly to a single, persistent gopls daemon.
package lsp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
)

// Position represents a 0-indexed position within a text document.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range represents a span inside a document.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location represents a coordinates range inside a target URI.
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// TextDocumentIdentifier targets a specific file URI.
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// DefinitionParams contains parameters for the textDocument/definition request.
type DefinitionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// Client represents the persistent gopls LSP JSON-RPC client.
type Client struct {
	conn      net.Conn
	dec       *json.Decoder
	enc       *json.Encoder
	mu        sync.Mutex
	idCounter int64
	pending   map[int64]chan *jsonResponse
	closeChan chan struct{}
}

type jsonRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type jsonResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonError      `json:"error,omitempty"`
}

type jsonError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewClientWithConn creates an LSP client using a pre-established net.Conn connection (ideal for testing).
func NewClientWithConn(conn net.Conn) *Client {
	c := &Client{
		conn:      conn,
		dec:       json.NewDecoder(conn),
		enc:       json.NewEncoder(conn),
		pending:   make(map[int64]chan *jsonResponse),
		closeChan: make(chan struct{}),
	}
	go c.readLoop()
	return c
}

func (c *Client) readLoop() {
	for {
		var resp jsonResponse
		if err := c.dec.Decode(&resp); err != nil {
			// Connection closed or EOF
			c.mu.Lock()
			for _, ch := range c.pending {
				close(ch)
			}
			c.pending = make(map[int64]chan *jsonResponse)
			c.mu.Unlock()
			return
		}

		c.mu.Lock()
		ch, ok := c.pending[resp.ID]
		if ok {
			delete(c.pending, resp.ID)
			ch <- &resp
		}
		c.mu.Unlock()
	}
}

// Initialize performs the standard initialize and initialized LSP handshake.
func (c *Client) Initialize(ctx context.Context) error {
	var result interface{}
	err := c.call(ctx, "initialize", map[string]interface{}{
		"processId":    0,
		"capabilities": map[string]interface{}{},
	}, &result)
	if err != nil {
		return err
	}

	return c.notify("initialized", map[string]interface{}{})
}

// GetDefinition retrieves definition coordinates (converting 1-indexed input to 0-indexed LSP queries).
func (c *Client) GetDefinition(ctx context.Context, filename string, line, col int) ([]Location, error) {
	var result []Location
	params := DefinitionParams{
		TextDocument: TextDocumentIdentifier{URI: "file://" + filename},
		Position: Position{
			Line:      line - 1,
			Character: col - 1,
		},
	}

	err := c.call(ctx, "textDocument/definition", params, &result)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("connection closed by language server")
		}
		return nil, err
	}
	return result, nil
}

// Close gracefully stops the LSP session by sending shutdown/exit sequence.
func (c *Client) Close(ctx context.Context) error {
	var result interface{}
	_ = c.call(ctx, "shutdown", nil, &result)
	_ = c.notify("exit", nil)
	return c.conn.Close()
}

func (c *Client) call(ctx context.Context, method string, params interface{}, result interface{}) error {
	c.mu.Lock()
	c.idCounter++
	id := c.idCounter
	req := jsonRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
	ch := make(chan *jsonResponse, 1)
	c.pending[id] = ch
	err := c.enc.Encode(req)
	c.mu.Unlock()

	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return ctx.Err()
	case resp, ok := <-ch:
		if !ok {
			return io.EOF
		}
		if resp.Error != nil {
			return fmt.Errorf("LSP error [%d]: %s", resp.Error.Code, resp.Error.Message)
		}
		if result != nil && len(resp.Result) > 0 {
			return json.Unmarshal(resp.Result, result)
		}
		return nil
	}
}

func (c *Client) notify(method string, params interface{}) error {
	c.mu.Lock()
	req := jsonRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	err := c.enc.Encode(req)
	c.mu.Unlock()
	return err
}
