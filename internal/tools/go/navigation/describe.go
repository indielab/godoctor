// Package navigation implements tools for navigating Go source code.
package navigation

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/danicat/godoctor/internal/lsp"
	"github.com/danicat/godoctor/internal/roots"
	"github.com/danicat/godoctor/internal/toolnames"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register registers the tool with the server.
func Register(server *mcp.Server) {
	def := toolnames.Registry["describe_symbol"]
	mcp.AddTool(server, &mcp.Tool{
		Name:        def.Name,
		Title:       def.Title,
		Description: def.Description,
	}, Handler)
}

// Params defines the input parameters for describe_symbol.
type Params struct {
	Filename string `json:"filename" jsonschema:"The absolute path to the Go file. Must be absolute."`
	Line     int    `json:"line" jsonschema:"The 1-indexed line number of the symbol"`
	Col      int    `json:"col" jsonschema:"The 1-indexed column number of the symbol"`
}

// Handler handles the describe_symbol tool execution.
func Handler(ctx context.Context, req *mcp.CallToolRequest, args Params) (*mcp.CallToolResult, any, error) {
	var session *mcp.ServerSession
	if req != nil {
		session = req.Session
	}
	absPath, err := roots.Global.Validate(session, args.Filename)
	if err != nil {
		return errorResult(err.Error()), nil, err
	}

	// Retrieve active connection to the persistent gopls language server
	client, err := lsp.GlobalManager.Client(ctx)
	if err != nil {
		return errorResult("failed to connect to language server: " + err.Error()), nil, err
	}

	definition, err := fetchDefinition(ctx, client, absPath, args.Line, args.Col)
	if err != nil {
		return errorResult("Failed to query symbol definition: " + err.Error()), nil, err
	}

	references := fetchReferences(ctx, absPath, args.Line, args.Col)

	// Format into Markdown
	var sb strings.Builder
	fmt.Fprintf(&sb, "## Symbol Description for `%s:%d:%d`\n\n", filepathBase(absPath), args.Line, args.Col)
	sb.WriteString("### Definition & Signature\n")
	sb.WriteString("```\n")
	sb.WriteString(definition)
	sb.WriteString("\n```\n\n")

	sb.WriteString("### Workspace References\n")
	sb.WriteString("```\n")
	sb.WriteString(references)
	sb.WriteString("\n```\n")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: sb.String()},
		},
	}, nil, nil
}

func fetchDefinition(ctx context.Context, client *lsp.Client, path string, line, col int) (string, error) {
	locs, err := client.GetDefinition(ctx, path, line, col)
	if err != nil {
		return "", err
	}
	if len(locs) == 0 {
		return "No definition found.", nil
	}
	loc := locs[0]
	return fmt.Sprintf(
		"URI: %s\nRange: %d:%d -> %d:%d",
		loc.URI,
		loc.Range.Start.Line+1,
		loc.Range.Start.Character+1,
		loc.Range.End.Line+1,
		loc.Range.End.Character+1,
	), nil
}

// nolint:gosec // G204: gopls is a trusted binary on the system path
func fetchReferences(ctx context.Context, path string, line, col int) string {
	position := fmt.Sprintf("%s:%d:%d", path, line, col)
	cmd := exec.CommandContext(ctx, "gopls", "references", position)
	refOut, refErr := cmd.CombinedOutput()

	if refErr != nil {
		return fmt.Sprintf("⚠️ Failed to find references: %s", strings.TrimSpace(string(refOut)))
	}
	references := strings.TrimSpace(string(refOut))
	if references == "" {
		return "No references found."
	}
	return references
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}

// Simple helper to avoid importing "path/filepath" unless necessary, or just extract basename.
func filepathBase(path string) string {
	idx := strings.LastIndexAny(path, "/\\")
	if idx == -1 {
		return path
	}
	return path[idx+1:]
}
