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
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
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
func isGoFile(input map[string]interface{}) bool {
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
func evaluateRunCommand(input map[string]interface{}) HookResponse {
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

	// 1. Build/Test Commands — always blocked: these are Go-specific by definition.
	buildPatterns := []string{"go build", "go test", "go vet", "golangci-lint"}
	for _, p := range buildPatterns {
		if strings.Contains(cmdStr, p) {
			return deny(
				"Quality Gate Hook: Manual toolchains are blocked. You MUST use GoDoctor's "+
					"`smart_build` tool to execute the quality gate pipeline "+
					"(tidy -> modernize -> format -> test -> lint).",
				"🛑 Blocked manual build/test",
			)
		}
	}

	// 2. Dependency Commands — always blocked: go get is Go-specific.
	if strings.HasPrefix(cmdStr, "go get") || strings.Contains(cmdStr, " go get ") {
		return deny(
			"Optimization Hook: Use `add_dependency` to install packages. It fetches the "+
				"documentation automatically, saving you a context-gathering step.",
			"🛑 Blocked go get",
		)
	}

	// 3. File Writers — only block when the command targets a .go file.
	if strings.Contains(cmdStr, ".go") {
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
		// echo redirecting into a .go file
		if strings.Contains(cmdStr, "echo ") && strings.Contains(cmdStr, ">") &&
			!strings.Contains(cmdStr, "> /dev/null") && !strings.Contains(cmdStr, ">/dev/null") {
			return deny(
				"Optimization Hook: Shell file modifications are blocked for Go files. "+
					"Use `smart_edit` to modify Go files safely.",
				"🛑 Blocked raw file write",
			)
		}
	}

	// 4. File Readers — only block when the command targets a .go file.
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

// handleShellCommand is kept for backward compatibility with direct callers.
func handleShellCommand(input map[string]interface{}) {
	_ = evaluateRunCommand(input)
	// We call writeResponse inside Intercept/evaluate, but handleShellCommand is kept for compatibility.
	resp := evaluateRunCommand(input)
	writeResponse(resp)
	os.Exit(0)
}
