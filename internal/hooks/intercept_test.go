package hooks

import (
	"testing"
)

func makePayload(toolName string, input map[string]interface{}) HookPayload {
	return HookPayload{
		ToolCall: ToolCall{
			Name: toolName,
			Args: input,
		},
	}
}

func pathInput(path string) map[string]interface{} {
	return map[string]interface{}{"path": path}
}

func absolutePathInput(path string) map[string]interface{} {
	return map[string]interface{}{"AbsolutePath": path}
}

func targetFileInput(path string) map[string]interface{} {
	return map[string]interface{}{"TargetFile": path}
}

func cmdInput(cmd string) map[string]interface{} {
	return map[string]interface{}{"command": cmd}
}

func cmdLineInput(cmd string) map[string]interface{} {
	return map[string]interface{}{"CommandLine": cmd}
}

// assertDeny checks the response is a deny and fails the test with a helpful message if not.
func assertDeny(t *testing.T, resp HookResponse, label string) {
	t.Helper()
	if resp.Decision != "deny" {
		t.Errorf("%s: expected deny, got %q (reason: %s)", label, resp.Decision, resp.Reason)
	}
}

// assertAllow checks the response is an allow and fails the test with a helpful message if not.
func assertAllow(t *testing.T, resp HookResponse, label string) {
	t.Helper()
	if resp.Decision != "allow" {
		t.Errorf("%s: expected allow, got %q (reason: %s)", label, resp.Decision, resp.Reason)
	}
}

// ---------------------------------------------------------------------------
// replace_file_content & multi_replace_file_content tools
// ---------------------------------------------------------------------------

func TestReplace_GoFile_Deny(t *testing.T) {
	// test replace_file_content
	assertDeny(t, evaluate(makePayload("replace_file_content", pathInput("main.go"))),
		"replace_file_content main.go with path")
	assertDeny(t, evaluate(makePayload("replace_file_content", targetFileInput("internal/server/handler.go"))),
		"replace_file_content subdir .go with TargetFile")

	// test multi_replace_file_content
	assertDeny(t, evaluate(makePayload("multi_replace_file_content", pathInput("main.go"))),
		"multi_replace_file_content main.go with path")
	assertDeny(t, evaluate(makePayload("multi_replace_file_content", targetFileInput("internal/server/handler.go"))),
		"multi_replace_file_content subdir .go with TargetFile")
}

func TestReplace_NonGoFile_Allow(t *testing.T) {
	for _, tool := range []string{"replace_file_content", "multi_replace_file_content"} {
		assertAllow(t, evaluate(makePayload(tool, pathInput("script.py"))), tool+" .py")
		assertAllow(t, evaluate(makePayload(tool, targetFileInput("README.md"))), tool+" .md")
		assertAllow(t, evaluate(makePayload(tool, absolutePathInput("config.yaml"))), tool+" .yaml")
		assertAllow(t, evaluate(makePayload(tool, pathInput("Dockerfile"))), tool+" Dockerfile")
	}
}

func TestReplace_NoPath_Allow(t *testing.T) {
	assertAllow(t, evaluate(makePayload("replace_file_content", map[string]interface{}{})), "replace_file_content no path")
	assertAllow(t, evaluate(makePayload("multi_replace_file_content", map[string]interface{}{})),
		"multi_replace_file_content no path")
}

// ---------------------------------------------------------------------------
// view_file tool (formerly read_file)
// ---------------------------------------------------------------------------

func TestViewFile_GoFile_Deny(t *testing.T) {
	assertDeny(t, evaluate(makePayload("view_file", pathInput("cmd/godoctor/main.go"))), "view_file main.go with path")
	assertDeny(t, evaluate(makePayload("view_file", absolutePathInput("types.go"))),
		"view_file types.go with AbsolutePath")
}

func TestViewFile_NonGoFile_Allow(t *testing.T) {
	assertAllow(t, evaluate(makePayload("view_file", absolutePathInput("README.md"))), "view_file .md")
	assertAllow(t, evaluate(makePayload("view_file", pathInput("package.json"))), "view_file .json")
	assertAllow(t, evaluate(makePayload("view_file", pathInput("index.ts"))), "view_file .ts")
	assertAllow(t, evaluate(makePayload("view_file", pathInput(".env"))), "view_file .env")
}

func TestViewFile_NoPath_Allow(t *testing.T) {
	assertAllow(t, evaluate(makePayload("view_file", map[string]interface{}{})), "view_file no path")
}

// ---------------------------------------------------------------------------
// write_to_file tool (formerly write_file)
// ---------------------------------------------------------------------------

func TestWriteToFile_GoFile_Deny(t *testing.T) {
	assertDeny(t, evaluate(makePayload("write_to_file", pathInput("new_service.go"))), "write_to_file .go with path")
	assertDeny(t, evaluate(makePayload("write_to_file", targetFileInput("pkg/util/helper.go"))),
		"write_to_file subdir .go with TargetFile")
}

func TestWriteToFile_NonGoFile_Allow(t *testing.T) {
	assertAllow(t, evaluate(makePayload("write_to_file", targetFileInput("index.html"))), "write_to_file .html")
	assertAllow(t, evaluate(makePayload("write_to_file", pathInput("styles.css"))), "write_to_file .css")
	assertAllow(t, evaluate(makePayload("write_to_file", pathInput("main.py"))), "write_to_file .py")
}

func TestWriteToFile_NoPath_Allow(t *testing.T) {
	assertAllow(t, evaluate(makePayload("write_to_file", map[string]interface{}{})), "write_to_file no path")
}

// ---------------------------------------------------------------------------
// run_command (formerly run_shell_command) — build/test (always blocked)
// ---------------------------------------------------------------------------

func TestShell_GoBuild_Deny(t *testing.T) {
	assertDeny(t, evaluate(makePayload("run_command", cmdInput("go build ./..."))), "go build ./... command")
	assertDeny(t, evaluate(makePayload("run_command", cmdLineInput("go build -o bin/app ."))), "go build -o CommandLine")
}

func TestShell_GoTest_Deny(t *testing.T) {
	assertDeny(t, evaluate(makePayload("run_command", cmdInput("go test ./..."))), "go test ./...")
	assertDeny(t, evaluate(makePayload("run_command", cmdLineInput("go test -v -run TestFoo ./pkg/..."))),
		"go test -v CommandLine")
}

func TestShell_GoVet_Deny(t *testing.T) {
	assertDeny(t, evaluate(makePayload("run_command", cmdInput("go vet ./..."))), "go vet")
}

func TestShell_GolangciLint_Deny(t *testing.T) {
	assertDeny(t, evaluate(makePayload("run_command", cmdInput("golangci-lint run"))), "golangci-lint run")
}

func TestShell_GoGet_Deny(t *testing.T) {
	assertDeny(t, evaluate(makePayload("run_command", cmdInput("go get github.com/pkg/errors"))), "go get")
	assertDeny(t, evaluate(makePayload("run_command", cmdLineInput("cd myapp && go get ./..."))),
		"inline go get CommandLine")
}

// ---------------------------------------------------------------------------
// run_command — file writers on .go files (blocked)
// ---------------------------------------------------------------------------

func TestShell_SedOnGoFile_Deny(t *testing.T) {
	assertDeny(t, evaluate(makePayload("run_command", cmdInput("sed -i 's/foo/bar/g' main.go"))), "sed -i .go")
}

func TestShell_TeeOnGoFile_Deny(t *testing.T) {
	assertDeny(t, evaluate(makePayload("run_command", cmdInput("echo 'package main' | tee main.go"))), "tee .go")
}

func TestShell_EchoRedirectToGoFile_Deny(t *testing.T) {
	assertDeny(t, evaluate(makePayload("run_command", cmdInput("echo 'package main' > main.go"))), "echo > .go")
	assertDeny(t, evaluate(makePayload("run_command", cmdLineInput("echo 'package main' >> main.go"))),
		"echo >> .go CommandLine")
}

// ---------------------------------------------------------------------------
// run_command — file writers on non-.go files (allowed)
// ---------------------------------------------------------------------------

func TestShell_SedOnNonGoFile_Allow(t *testing.T) {
	assertAllow(t, evaluate(makePayload("run_command", cmdInput("sed -i 's/foo/bar/g' config.yaml"))), "sed -i .yaml")
	assertAllow(t, evaluate(makePayload("run_command", cmdLineInput("sed -i 's/v1/v2/' Dockerfile"))),
		"sed -i Dockerfile CommandLine")
}

func TestShell_EchoRedirectToNonGoFile_Allow(t *testing.T) {
	assertAllow(t, evaluate(makePayload("run_command", cmdInput("echo 'hello' > output.txt"))), "echo > .txt")
}

func TestShell_EchoToDevNull_Allow(t *testing.T) {
	assertAllow(t, evaluate(makePayload("run_command", cmdInput("echo 'test' > /dev/null"))), "echo > /dev/null")
}

// ---------------------------------------------------------------------------
// run_command — file readers on .go files (blocked)
// ---------------------------------------------------------------------------

func TestShell_CatGoFile_Deny(t *testing.T) {
	assertDeny(t, evaluate(makePayload("run_command", cmdInput("cat main.go"))), "cat main.go")
	assertDeny(t, evaluate(makePayload("run_command", cmdLineInput("cat internal/hooks/intercept.go"))),
		"cat subdir .go CommandLine")
}

func TestShell_GrepGoFile_Allow(t *testing.T) {
	assertAllow(t, evaluate(makePayload("run_command", cmdInput("grep 'func ' main.go"))), "grep .go")
}

// ---------------------------------------------------------------------------
// run_command — file readers on .go files with pipes (allowed)
// ---------------------------------------------------------------------------

func TestShell_GrepGoFileWithPipe_Allow(t *testing.T) {
	assertAllow(t, evaluate(makePayload("run_command", cmdInput("grep 'TODO' main.go | wc -l"))), "grep .go | pipe")
}

// ---------------------------------------------------------------------------
// run_command — file readers on non-.go files (allowed)
// ---------------------------------------------------------------------------

func TestShell_CatNonGoFile_Allow(t *testing.T) {
	assertAllow(t, evaluate(makePayload("run_command", cmdInput("cat README.md"))), "cat .md")
	assertAllow(t, evaluate(makePayload("run_command", cmdLineInput("cat package.json"))), "cat .json CommandLine")
}

func TestShell_GrepNonGoFile_Allow(t *testing.T) {
	assertAllow(t, evaluate(makePayload("run_command", cmdInput("grep 'error' server.log"))), "grep .log")
}

// ---------------------------------------------------------------------------
// run_command — safe commands (always allowed)
// ---------------------------------------------------------------------------

func TestShell_SafeCommands_Allow(t *testing.T) {
	cases := []struct {
		label string
		cmd   string
	}{
		{"npm install", "npm install"},
		{"pip install", "pip install requests"},
		{"ls", "ls -la"},
		{"mkdir", "mkdir -p dist"},
		{"git status", "git status"},
		{"curl", "curl https://example.com"},
		{"docker build", "docker build -t myapp ."},
		{"make", "make build"},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			assertAllow(t, evaluate(makePayload("run_command", cmdInput(tc.cmd))), tc.label+" command")
			assertAllow(t, evaluate(makePayload("run_command", cmdLineInput(tc.cmd))), tc.label+" CommandLine")
		})
	}
}

// ---------------------------------------------------------------------------
// Unknown tools (always allowed)
// ---------------------------------------------------------------------------

func TestUnknownTool_Allow(t *testing.T) {
	assertAllow(t, evaluate(makePayload("search_web", map[string]interface{}{"query": "golang"})), "search_web")
	assertAllow(t, evaluate(makePayload("list_dir", map[string]interface{}{"path": "/tmp"})), "list_dir")
	assertAllow(t, evaluate(makePayload("", map[string]interface{}{})), "empty tool name")
}
