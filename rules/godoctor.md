# GoDoctor Agent Rule & Usage Instructions

GoDoctor is a specialized and optimized suite of tools and skills carefully engineered to elevate agentic coding in Go codebases.

> [!IMPORTANT]
> **MANDATORY FOR ALL AGENTS WORKING ON GO CODE**:
> Whenever GoDoctor is installed, coding agents operating on Go codebases (`.go` files, `go.mod`, Go toolchains) **MUST** use GoDoctor tools (`smart_build`, `smart_edit`, `smart_read`, `describe_symbol`, `add_dependency`, `read_docs`, `mutation_test`, `test_query`, `list_files`) instead of generic shell commands or unverified raw file tools.

---

## Core Mandates

1. **Mandatory Specialized Tooling**:
   - Use `smart_build` for ALL compilation, package testing, formatting, and linting tasks. Do NOT run manual `go build` or `go test` shell commands.
   - Use `smart_edit` for ALL file modifications. Edits are atomically checked via `gopls` type checking before writing to disk.
   - Use `smart_read` for reading source files. It automatically enriches snippets with `<types>` blocks displaying struct and interface definitions.
   - Use `describe_symbol` to inspect symbol signatures, declaration coordinates, and workspace call-sites instantly.
2. **Context & Symbol Exploration**: Before editing Go code, inspect structural outlines and symbol definitions (`smart_read`, `describe_symbol`).
3. **Atomic Safety & Rollback**: Edits with `smart_edit` automatically apply `gofmt`/`goimports` and verify type safety. If compilation fails, edits are safely rolled back to prevent on-disk corruption.
4. **Idiomatic Go**: Follow Go team best practices (`go-standards`, `go-audit`). Reject legacy `pkg/` folders, enterprise package bloat, premature interfaces, and package stuttering.

---

## Tool Reference

### 🔍 Navigation & Discovery
- **`list_files`**: Lists source files while excluding version control directories (`.git`).
- **`smart_read`**: Structure-aware Go source code reader.
  - **Type Enrichment**: Appends exact struct/interface definitions of referenced types in `<types>` blocks.
  - **Snippet Mode**: Target specific line ranges (`start_line`, `end_line`).
  - **Outline Mode (`outline=true`)**: Retrieve structural AST outlines.
- **`describe_symbol`**: Queries `gopls` for exact declaration signatures, package comments, line coordinates, and workspace call-sites for any symbol.

### ✏️ Specialized Editing
- **`smart_edit`**: Atomic, compiler-gated multi-file editor.
  - **Compilation Check**: Runs `gofmt`, `goimports`, and `gopls check ./...` *before* committing edits.
  - **Automated Rollback**: If syntax or type errors occur, edits are completely rolled back and Levenshtein spelling suggestions are returned.
  - **Line Isolation**: Use `start_line` and `end_line` for precise, unambiguous block targeting.

### 🛠️ Go Toolchain Integration
- **`smart_build`**: GoDoctor's specialized build pipeline (`go mod tidy` -> modernization -> `gofmt` -> `go build` -> `go test` -> linter).
- **`add_dependency`**: Installs Go modules and fetches API documentation automatically.
- **`read_docs`**: Instantly fetches package and symbol documentation from Go doc servers.

### 🧪 Quality Engineering & Testing
- **`mutation_test`**: Runs Selene mutation tests to inject code mutations (flipped conditions, operator swaps) to objectively verify that test suites detect behavioral regressions.
- **`test_query`**: Queries test results and coverage data using SQL via `testquery.db`. Find cold code paths (`SELECT * FROM all_coverage WHERE count = 0`), panics, and failing tests.

---

## Workflow Examples

### Feature Implementation:
1. `list_files` to map project structure.
2. `smart_read` (Outline mode) or `describe_symbol` to inspect type signatures and entry points.
3. `smart_edit` to implement changes or append new functions.
4. `smart_build` to run complete compilation, tests, and formatting.

### Bug Fix & Test Engineering:
1. `smart_edit` to append a new table-driven unit test in `_test.go`.
2. `smart_build` to verify test reproduction.
3. `smart_read` to inspect implementation logic.
4. `smart_edit` to fix the implementation.
5. `smart_build` to confirm test pass.
6. `test_query` or `mutation_test` to verify code coverage and assertion strength.
