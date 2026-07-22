// Package hooks implements pre-invocation interception rules for Antigravity tools.
package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Const definitions to satisfy goconst and clean code.
const (
	decisionDeny = "deny"
	keyPath      = "path"
)

// HookPayload represents the JSON payload sent by Antigravity CLI to the hook via stdin.
type HookPayload struct {
	ToolCall              ToolCall `json:"toolCall"`
	StepIdx               int      `json:"stepIdx"`
	ConversationID        string   `json:"conversationId"`
	WorkspacePaths        []string `json:"workspacePaths"`
	TranscriptPath        string   `json:"transcriptPath"`
	ArtifactDirectoryPath string   `json:"artifactDirectoryPath"`
}

// ToolCall represents the actual tool invocation being checked.
type ToolCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// HookResponse represents the decision returned to Antigravity CLI via stdout.
type HookResponse struct {
	Decision      string `json:"decision"`
	Reason        string `json:"reason,omitempty"`
	SystemMessage string `json:"systemMessage,omitempty"`
}

// Intercept reads the tool payload from standard input, evaluates it against the rules,
// and outputs the decision to standard output.
func Intercept() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		writeResponse(HookResponse{
			Decision:      decisionDeny,
			Reason:        "Failed to read hook payload: " + err.Error(),
			SystemMessage: "🛑 Error",
		})
		os.Exit(0)
		return
	}

	var payload HookPayload
	if err := json.Unmarshal(input, &payload); err != nil {
		writeResponse(HookResponse{
			Decision:      decisionDeny,
			Reason:        "Failed to parse hook payload: " + err.Error(),
			SystemMessage: "🛑 Parse Error",
		})
		os.Exit(0)
		return
	}

	resp := evaluate(payload)
	writeResponse(resp)
	os.Exit(0)
}

// evaluate applies the interception rules to a payload and returns the decision.
// It is pure (no I/O, no os.Exit) so that it can be unit-tested directly.
func evaluate(payload HookPayload) HookResponse {
	switch payload.ToolCall.Name {
	case "replace_file_content", "multi_replace_file_content":
		if !isGoFile(payload.ToolCall.Args) {
			return allow()
		}
		return deny(
			"Optimization Hook: The native `replace_file_content` and `multi_replace_file_content` "+
				"tools are blocked for Go files. You MUST use GoDoctor's `smart_edit` tool for safe, "+
				"fuzzy-matched, syntax-verified file modifications.",
			"🛑 Blocked raw replace",
		)
	case "view_file":
		if !isGoFile(payload.ToolCall.Args) {
			return allow()
		}
		return deny(
			"Optimization Hook: Raw reads are blocked for Go files. You MUST use GoDoctor's "+
				"`smart_read` to inspect Go code. It provides structural outlines and context-aware scoping.",
			"🛑 Blocked raw read",
		)
	case "write_to_file":
		if !isGoFile(payload.ToolCall.Args) {
			return allow()
		}
		return deny(
			"Optimization Hook: Raw file creation is blocked for Go files. You MUST use GoDoctor's "+
				"`smart_edit` tool which handles atomic file creation natively.",
			"🛑 Blocked raw write",
		)
	case "run_command":
		return evaluateRunCommand(payload.ToolCall.Args)
	}

	return allow()
}

// isGoFile reports whether the tool input's path targets a Go source file.
func isGoFile(input map[string]any) bool {
	for _, key := range []string{"AbsolutePath", "TargetFile", keyPath, "file_path", "filename"} {
		if val, ok := input[key]; ok {
			if s, ok := val.(string); ok {
				return strings.HasSuffix(s, ".go")
			}
		}
	}
	return false
}

// evaluateRunCommand parses and checks a command line string for blocked patterns.
func evaluateRunCommand(input map[string]any) HookResponse {
	cmdInterface, ok := input["CommandLine"]
	if !ok {
		cmdInterface, ok = input["command"]
		if !ok {
			return allow()
		}
	}

	cmdStr, ok := cmdInterface.(string)
	if !ok {
		return allow()
	}

	cmdStr = strings.TrimSpace(cmdStr)

	if resp := checkBuildTestCommands(cmdStr); resp.Decision == decisionDeny {
		return resp
	}
	if resp := checkDependencyCommands(cmdStr); resp.Decision == decisionDeny {
		return resp
	}
	if resp := checkFileWriters(cmdStr); resp.Decision == decisionDeny {
		return resp
	}
	if resp := checkFileReaders(cmdStr); resp.Decision == decisionDeny {
		return resp
	}
	return allow()
}

func checkBuildTestCommands(cmdStr string) HookResponse {
	buildPatterns := []string{"go build", "go test", "go vet", "golangci-lint"}
	for _, p := range buildPatterns {
		if strings.Contains(cmdStr, p) {
			return deny(
				"GoDoctor Mandate: Manual toolchains are blocked. GoDoctor is a specialized "+
					"and optimized tool suite created to improve agentic coding in Go. You MUST "+
					"use GoDoctor's `smart_build` tool for Go build, test, tidying, formatting, "+
					"and linting workflows.",
				"🛑 Blocked manual build/test",
			)
		}
	}
	return allow()
}

func checkDependencyCommands(cmdStr string) HookResponse {
	if strings.HasPrefix(cmdStr, "go get") || strings.Contains(cmdStr, " go get ") {
		return deny(
			"Optimization Hook: Use `add_dependency` to install packages. It fetches the "+
				"documentation automatically, saving you a context-gathering step.",
			"🛑 Blocked go get",
		)
	}
	return allow()
}

func checkFileWriters(cmdStr string) HookResponse {
	if !strings.Contains(cmdStr, ".go") {
		return allow()
	}
	writePatterns := []string{"sed -i", "tee "}
	for _, p := range writePatterns {
		if strings.Contains(cmdStr, p) {
			return deny(
				"Optimization Hook: Shell file modifications are blocked for Go files. "+
					"Use `smart_edit` to modify Go files safely.",
				"🛑 Blocked raw file edit",
			)
		}
	}
	if strings.Contains(cmdStr, "echo ") && strings.Contains(cmdStr, ">") &&
		!strings.Contains(cmdStr, "> /dev/null") && !strings.Contains(cmdStr, ">/dev/null") {
		return deny(
			"Optimization Hook: Shell file modifications are blocked for Go files. "+
				"Use `smart_edit` to modify Go files safely.",
			"🛑 Blocked raw file write",
		)
	}
	return allow()
}

func checkFileReaders(cmdStr string) HookResponse {
	if strings.HasPrefix(cmdStr, "cat ") && strings.Contains(cmdStr, ".go") ||
		strings.Contains(cmdStr, " cat ") && strings.Contains(cmdStr, ".go") {
		return deny(
			"Optimization Hook: Raw shell reads are blocked for Go files. "+
				"You MUST use GoDoctor's `smart_read` to inspect Go code.",
			"🛑 Blocked shell cat",
		)
	}
	return allow()
}

func allow() HookResponse {
	return HookResponse{Decision: "allow"}
}

func deny(reason, systemMessage string) HookResponse {
	return HookResponse{
		Decision:      decisionDeny,
		Reason:        reason,
		SystemMessage: systemMessage,
	}
}

func writeResponse(resp HookResponse) {
	out, _ := json.Marshal(resp)
	fmt.Println(string(out))
}
