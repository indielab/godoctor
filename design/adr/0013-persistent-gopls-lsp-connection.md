# ADR-0013: Persistent gopls LSP Connection

- **Status:** Approved
- **Date:** 2026-06-01
- **Author(s):** Daniela Petruzalek
- **Deciders:** Daniela Petruzalek, Claude Opus 4.6, Antigravity

## 1. Context
GoDoctor's code discovery and navigation features (such as type-enrichment in `smart_read` and declaration lookups in `describe_symbol`) previously relied on spawning short-lived, short-circuit external CLI commands (e.g., executing `gopls definition` or `gopls symbols`). 

Spawning external CLI processes for every single AST coordinate lookup introduced severe performance bottlenecks:
- Every execution incurred the full overhead of starting a brand-new `gopls` JVM/Go process.
- Each subprocess had to load, parse, and re-index the workspace files from scratch.
- Query latencies climbed past several seconds when resolving multiple coordinates concurrently.

To achieve fast, responsive tool feedback loops for calling agents, GoDoctor needed a way to execute coordinate type-lookups in milliseconds rather than seconds.

## 2. Decision
We decided to implement a stateful, persistent Language Server Protocol (LSP) connection within GoDoctor, transitioning fully away from individual CLI subprocesses. 

Specifically:
- We created a unified `internal/lsp` package containing a stateful client (`client.go`) and a background daemon process manager (`manager.go`).
- The manager spawns and maintains a single shared `gopls serve` daemon process per workspace session.
- Communication with `gopls` is conducted via multiplexed JSON-RPC over stdin/stdout, allowing parallel non-blocking requests.
- The `smart_read` and `describe_symbol` tools retrieve this shared background connection to resolve AST types, definitions, and references instantly.
- Added a graceful teardown handler in `internal/server/server.go` to stop the persistent daemon when the MCP server shuts down.

## 3. Consequences
- **Positive:** Reduces type resolution and definition coordinates lookup latency from seconds to milliseconds. Resolves multiple AST type specifications concurrently with negligible performance overhead.
- **Negative:** Introduces stateful background process lifecycle management. GoDoctor must correctly track process status and recover from any underlying `gopls` crash or connection issue.
- **Neutral:** Clean, mocked TDD client test coverage verifies connection protocols, handshakes, and shutdowns safely without requiring a real binary setup during local unit tests.
