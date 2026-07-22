package lsp_test

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"testing"
	"time"

	"github.com/danicat/godoctor/internal/lsp"
)

const jsonrpcVersion = "2.0"
const resultField = "result"
const jsonrpcField = "jsonrpc"

// mockServer mocks basic LSP Handshake behaviors and textDocument/definition queries
type mockServer struct {
	conn net.Conn
}

func (s *mockServer) serve(t *testing.T) {
	defer func() {
		_ = s.conn.Close()
	}()
	dec := json.NewDecoder(s.conn)
	enc := json.NewEncoder(s.conn)

	for {
		var msg struct {
			ID     interface{}     `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := dec.Decode(&msg); err != nil {
			if err == io.EOF {
				return
			}
			t.Errorf("mockServer decode error: %v", err)
			return
		}

		switch msg.Method {
		case "initialize":
			resp := map[string]interface{}{
				jsonrpcField: jsonrpcVersion,
				"id":         msg.ID,
				resultField: map[string]interface{}{
					"capabilities": map[string]interface{}{
						"definitionProvider": true,
					},
				},
			}
			_ = enc.Encode(resp)
		case "initialized":
			// Notification, no response
		case "textDocument/definition":
			resp := map[string]interface{}{
				jsonrpcField: jsonrpcVersion,
				"id":         msg.ID,
				resultField: []map[string]interface{}{
					{
						"uri": "file:///workspace/main.go",
						"range": map[string]interface{}{
							"start": map[string]interface{}{"line": 9, "character": 4},
							"end":   map[string]interface{}{"line": 9, "character": 12},
						},
					},
				},
			}
			_ = enc.Encode(resp)
		case "shutdown":
			resp := map[string]interface{}{
				jsonrpcField: jsonrpcVersion,
				"id":         msg.ID,
				resultField:  nil,
			}
			_ = enc.Encode(resp)
		case "exit":
			return
		}
	}
}

func TestLSPClientTDD(t *testing.T) {
	// 1. Create fully in-memory pipe connection for reliable mocking
	cliConn, srvConn := net.Pipe()

	server := &mockServer{conn: srvConn}
	go server.serve(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 2. Initialize our LSP client wrapping the raw pipe
	client := lsp.NewClientWithConn(cliConn)

	// 3. Phase 1: Assert successful connection handshake
	err := client.Initialize(ctx, []string{"/workspace"})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// 4. Phase 2: Assert robust translation of coordinates into 0-indexed LSP queries
	locs, err := client.GetDefinition(ctx, "/workspace/main.go", 10, 5)
	if err != nil {
		t.Fatalf("GetDefinition failed: %v", err)
	}

	if len(locs) != 1 {
		t.Fatalf("Expected exactly 1 definition location, got %d", len(locs))
	}

	expectedURI := "file:///workspace/main.go"
	if locs[0].URI != expectedURI {
		t.Errorf("Expected URI %q, got %q", expectedURI, locs[0].URI)
	}
	if locs[0].Range.Start.Line != 9 {
		t.Errorf("Expected line index 9 (0-indexed line 10), got %d", locs[0].Range.Start.Line)
	}

	// 5. Phase 3: Assert clean lifecycle shutdown
	err = client.Close(ctx)
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestPathToURI(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"empty", "", ""},
		{"macOS/Linux absolute", "/usr/local/bin/go", "file:///usr/local/bin/go"},
		{"Windows drive letter", "C:\\Users\\User\\project", "file:///C:/Users/User/project"},
		{"Windows forward slash", "C:/Users/User/project", "file:///C:/Users/User/project"},
		{"Linux root", "/", "file:///"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lsp.PathToURI(tt.path)
			if got != tt.expected {
				t.Errorf("PathToURI(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

func TestPrematureSyncRoots(t *testing.T) {
	cliConn, srvConn := net.Pipe()

	client := lsp.NewClientWithConn(cliConn)
	defer func() { _ = client.Close(context.Background()) }()
	defer func() { _ = srvConn.Close() }()

	// Call SyncRoots before Initialize
	err := client.SyncRoots(context.Background(), []string{"/workspace/test"})
	if err != nil {
		t.Fatalf("Expected nil error (discarded), got %v", err)
	}
}
