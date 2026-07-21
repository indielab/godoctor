---
name: go-standards
description: >
  Use this skill to enforce official Go team development standards, project layout conventions, and idiomatic production patterns. Activate when initializing Go modules, structuring Go codebases, designing package boundaries, or reviewing Go code. Strictly enforces flat layout defaults for simple services and cmd/ & internal/ for complex apps while explicitly rejecting widely adopted antipatterns such as /pkg directories, enterprise subfolder bloat, premature interfaces, package stuttering, global mutable state, and panic error handling.
---

# Go Development Standards & Best Practices

This skill provides comprehensive instructions, guidelines, and production standards for designing, structuring, and developing modern Go modules (Go 1.24+). It enforces official Go team practices, promotes compiler-backed safety, and explicitly rejects common antipatterns and over-engineering traps.

---

## 1. Layout Conventions & Official Go Standards

Always adhere strictly to official [go.dev/doc/modules/layout](https://go.dev/doc/modules/layout) conventions.

### Flat Layout (Recommended Default for Simple Services & Libraries)
Keep all Go source files in the root directory. This is the official Go recommendation for simple microservices, tools, and libraries. It avoids unnecessary abstraction layers, eliminates package stuttering, and prevents import cycles.

```
my-service/
├── go.mod
├── go.sum
├── main.go
├── server.go
├── database.go
├── user.go
└── server_test.go
```
*Rule of thumb:* Default to flat layout. Do not introduce nested directories until scale, multiple binaries, or private package boundaries explicitly require it.

### Nested Layout (`cmd/` & `internal/`)
Use a nested layout only when building multiple executable binaries or protecting private package implementations from external module imports.

```
my-app/
├── go.mod
├── go.sum
├── cmd/
│   ├── api-server/
│   │   └── main.go
│   └── cli-tool/
│       └── main.go
└── internal/
    ├── auth/
    │   └── auth.go
    └── db/
        └── db.go
```
- **`cmd/`**: Contains subdirectories for each executable binary. Keep code in `cmd/<binary>/main.go` minimal (flag parsing, dependency wiring, and calling `run()`).
- **`internal/`**: Contains private application logic and packages. The Go toolchain enforces import boundaries; code outside this module cannot import anything under `internal/`.

---

## 2. Explicit Antipattern Rejection (DO NOT DO)

### ❌ `pkg/` Directory
- **Antipattern**: Creating a root-level `pkg/` directory (e.g., `my-project/pkg/user`).
- **Why Reject**: `pkg/` is a legacy/non-standard directory pattern, explicitly rejected in modern Go standards. It adds redundant path depth without adding any access control (unlike `internal/`).

### ❌ Enterprise Package Bloat / Clean-Hexagonal Over-Engineering
- **Antipattern**: Creating `adapters/`, `ports/`, `entities/`, `controllers/`, `repositories/`, `services/`, or `usecases/` folders for simple Go services.
- **Why Reject**: Translating Java/C# enterprise patterns into Go results in deep directory hierarchies, massive boilerplate, circular import workarounds, and unidiomatic code. In Go, group by domain or feature (or keep flat), not by architectural layer.

### ❌ Interface Bloat / Premature Abstraction
- **Antipattern**: Defining interfaces alongside concrete struct implementations before they are needed, or creating single-implementation interfaces in the provider package.
- **Why Reject**: Go interfaces are satisfied implicitly. Provider packages should return concrete types (`structs`). Interfaces MUST be defined by the **consumer** package that requires the behavior.
  - **Rule**: *Accept interfaces, return structs.*

### ❌ Package Stuttering
- **Antipattern**: Repeating package names in type or function names (`user.UserService` ❌, `db.DBClient` ❌, `config.ConfigStruct` ❌).
- **Why Reject**: Exported symbols are qualified by their package name at call-sites (`user.Service`, `db.Client`, `config.Config`).

### ❌ Package `utils` or `common`
- **Antipattern**: Creating catch-all packages like `utils`, `helpers`, `common`, or `shared`.
- **Why Reject**: These packages become dumping grounds for unrelated utility logic, degrade domain boundaries, and frequently lead to circular dependencies. Group by responsibility or keep near call-site.

### ❌ Global Mutable State / Singletons
- **Antipattern**: Package-level mutable variables (`var DB *sql.DB`, `var Config AppConfig`, `var Logger *log.Logger`).
- **Why Reject**: Global state makes unit testing impossible to run in parallel, introduces data races, and hides runtime dependencies. Pass dependencies explicitly via constructors.

### ❌ Panic for Control Flow / Error Swallowing
- **Antipattern**: Using `panic()` in library or service code, or ignoring returned errors with `_`.
- **Why Reject**: `panic` halts the process and degrades reliability. Always handle errors and wrap them with descriptive context using `fmt.Errorf("doing X: %w", err)`.

### ❌ Context Swallowing
- **Antipattern**: Ignoring `ctx context.Context` in network/database calls or calling `context.Background()` deep inside HTTP request handlers.
- **Why Reject**: Prevents request cancellation propagation, causes resource leaks, and breaks distributed tracing and timeouts.

---

## 3. Production Patterns (Go 1.24+)

### HTTP Server with `http.NewServeMux` & Graceful Shutdown (`signal.NotifyContext`)
Use standard library HTTP features (`http.NewServeMux` method matching) and handle process signals gracefully.

```go
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		log.Println("Shutting down HTTP server gracefully...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}
```

### Database Connection Pooling (`database/sql`)
Properly configure connection pool limits and verify connectivity on startup.

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func initDB(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database connection: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(3 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return db, nil
}
```

---

## 4. Go Development Workflow

When initializing or refactoring Go projects using GoDoctor tools:

1. **Initialize Module**: Run `project_init` to initialize the project directory and `go mod init <module_path>`.
2. **Choose Layout**: Decide between **Flat Layout** (default for single services/libraries) or **Nested Layout** (`cmd/` and `internal/` for multi-binary apps).
3. **Implement Code**: Use `smart_edit` to create or update Go source files, adhering to the antipattern rules above.
4. **Manage Dependencies**: Use `add_dependency` to query and add verified Go module dependencies.
5. **Build & Validate**: Execute `smart_build` to run GoDoctor's build pipeline (`go mod tidy` -> modernization -> `gofmt` -> `go build` -> `go test` -> linter).
