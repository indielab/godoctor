---
name: go-audit
description: >
  Use this skill to audit, diagnose, modernize, and engineer test suites for Go codebases. Activate whenever catching bugs, writing unit or integration tests, modernizing legacy Go code to Go 1.24+ standards, running mutation tests (mutation_test), or analyzing test coverage with SQL (test_query). Enforces table-driven unit tests, compiler-gated diagnostic workflows (smart_build, smart_read, describe_symbol), cold-path coverage analysis, and assertion strength verification.
---

# Go Audit & Quality Engineering

GoDoctor is a specialized and optimized suite of tools and skills carefully engineered to elevate agentic coding in Go codebases. This skill details the workflows and diagnostics required to audit codebases, catch bugs, modernize legacy code, and continuously improve code quality and testability.

---

## 1. The Go Quality & Audit Philosophy

Coding agents operating on Go codebases MUST prioritize high testability, zero compiler diagnostics, and robust behavioral coverage:

- **Catch Bugs Early**: Use compiler-gated diagnostics and type enrichment (`smart_read`, `describe_symbol`) to catch logic errors before executing code.
- **Modernize Code Idiomatically**: Upgrade legacy Go code to modern Go 1.24+ standards (e.g. `http.NewServeMux` routing, `log/slog`, `net/netip`, generics, type aliases).
- **Comprehensive Testability**: Write table-driven unit tests, run integration test suites with coverage (`smart_build`), analyze cold paths using SQL (`test_query`), and verify assertion strength via mutation testing (`mutation_test`).

---

## 2. Bug Isolation & Audit Diagnostics

Follow this 5-step diagnostic sequence to systematically isolate and correct Go issues:

### Step 1: Analyze Compilation Diagnostics
- Execute `smart_build` or inspect compilation feedback from `smart_edit`.
- Extract exact file paths, line numbers, and column coordinates (e.g., `main.go:34:12`).
- Categorize the issue: syntax error, missing import/dependency, or strict type mismatch.

### Step 2: Leverage Code & Symbol Queries
- Do not guess type definitions or method signatures.
- **`smart_read`**: Read source files with automatic type-tag annotations. The embedded `<types>` block displays full struct definitions, interface contracts, and field types of referenced symbols.
- **`describe_symbol`**: Query specific symbol coordinates (`line:col`) to inspect its exact declaration, package origin, method set, and workspace call-sites.

### Step 3: Check Spelling & Field Matching
- When encountering `undefined`, `undeclared name`, or `has no field or method`:
  - Review Levenshtein spelling suggestions returned by `smart_edit` (powered by `gopls symbols`).
  - Verify export visibility (Go capitalization rules: `Name` is exported, `name` is unexported).

### Step 4: Investigate Runtime & Coverage Gaps
- When code compiles but tests fail or runtime behavior is unexpected:
  - **`test_query`**: Execute SQL queries against `testquery.db` to isolate uncovered code paths (`SELECT * FROM all_coverage WHERE count = 0`), panics, or historic test failures.
  - **`mutation_test`**: Run mutation testing to locate weak test assertions or missing edge-case coverage by identifying surviving mutants.

### Step 5: Implement Atomic Corrections
- Apply all file changes using `smart_edit`.
- `smart_edit` verifies compilation via `gopls check ./...` before writing. If compilation fails, edits are safely rolled back to prevent workspace corruption.

---

## 3. Code Modernization (Go 1.24+)

Continuously modernize legacy Go codebases to leverage modern standard library capabilities:

- **HTTP Routing**: Replace third-party routers or legacy `http.HandleFunc` with modern `http.NewServeMux` path and method matching (`mux.HandleFunc("GET /items/{id}", handler)`).
- **Structured Logging**: Standardize logging on `log/slog` rather than untyped `log.Printf` or third-party loggers where appropriate.
- **Context Propagation**: Pass `context.Context` as the first argument in I/O operations and respect cancellation signals (`ctx.Done()`).

---

## 4. Testability & Test Suite Engineering

### Unit Test Creation
- **Table-Driven Tests**: Write idiomatic table-driven unit tests for functions with multiple inputs/outputs:
  ```go
  func TestCalculate(t *testing.T) {
      tests := []struct {
          name     string
          input    int
          expected int
          wantErr  bool
      }{
          {name: "valid positive", input: 5, expected: 10, wantErr: false},
          {name: "zero value", input: 0, expected: 0, wantErr: false},
      }
      for _, tt := range tests {
          t.Run(tt.name, func(t *testing.T) {
              got, err := Calculate(tt.input)
              if (err != nil) != tt.wantErr {
                  t.Errorf("Calculate(%d) error = %v, wantErr %v", tt.input, err, tt.wantErr)
                  return
              }
              if got != tt.expected {
                  t.Errorf("Calculate(%d) = %d, want %d", tt.input, got, tt.expected)
              }
          })
      }
  }
  ```

### Integration Testing & Coverage (`smart_build`)
- Run `smart_build` to execute GoDoctor's build pipeline and run package test suites with test coverage enabled across all packages (`./...`).

### SQL Coverage Analysis (`test_query`)
- Use `test_query` to query the `testquery.db` SQLite database to find cold code paths, test failures, and coverage gaps.
- **Empirically Verified Database Schemas**:
  - `all_tests`: Individual test run results.
    - Schema: `(time TIMESTAMP, action TEXT, package TEXT, test TEXT, elapsed NUMERIC, output TEXT)`
    - Example: `SELECT test, package, output FROM all_tests WHERE action = 'fail'`
  - `all_coverage`: Function and line statement coverage counts.
    - Schema: `(package TEXT, file TEXT, start_line INT, start_col INT, end_line INT, end_col INT, stmt_num INT, count INT, function_name TEXT)`
    - Example: `SELECT file, function_name, start_line, end_line FROM all_coverage WHERE count = 0 ORDER BY file`
  - `test_coverage`: Per-test granular line coverage map.
    - Schema: `(test_name TEXT, package TEXT, file TEXT, start_line INT, start_col INT, end_line INT, end_col INT, stmt_num INT, count INT, function_name TEXT)`
    - Example: `SELECT test_name, file, start_line FROM test_coverage WHERE count > 0`
  - `all_code`: Line-by-line source code index.
    - Schema: `(package TEXT, file TEXT, line_number INT, content TEXT)`
    - Example: `SELECT file, line_number, content FROM all_code WHERE content LIKE '%panic%'`
  - `metadata`: Database metadata.
    - Schema: `(key TEXT, value TEXT)`

### Mutation Testing (`mutation_test`)
- Execute `mutation_test` using Selene to introduce subtle code mutations (swapped operators, inverted conditionals).
- Ensure existing test suites kill mutants. If mutants survive, add specific unit tests covering those missing boundary conditions.

---

## 5. Gotchas & Common Pitfalls

- **Unexported Symbol Access**: Accessing lowercase package-level identifiers across package boundaries fails compilation. Verify symbol visibility.
- **Nil Interface Value Trap**: An interface holding a nil concrete pointer is non-nil (`err != nil` evaluates to true). Always return explicit `nil` interface values on success.
- **Shadowed Variables**: Watch for short variable declarations (`:=`) in inner blocks shadowing outer scope variables.
- **Unused Imports or Variables**: The Go compiler strictly rejects unused declarations. `smart_edit` and `smart_build` will reject files containing unused imports or variables.
