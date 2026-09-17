# Phase 8 Implementation Guide: Server Entry Point & Graceful Shutdown

This guide walks you step-by-step through implementing the production-ready entry point and graceful shutdown mechanism in **Phase 8** inside `cmd/api/main.go`.

Following our educational format, this document explains the mechanics of operating system signals, non-blocking HTTP listeners, context deadlines, and sequential resource teardown before you write the code yourself.

---

## 1. Architectural Overview & Why Graceful Shutdown Matters

When a Go application runs in production (e.g., inside Docker, Kubernetes, systemd, or local terminal), shutting it down abruptly (like pressing `Ctrl + C` or receiving `SIGTERM` during a rolling deployment) will immediately kill the process if not handled.

Abrupt termination causes serious issues:
- **In-flight HTTP requests are severed**: Clients mid-way through receiving a response or writing a post get connection drops or 502/504 Bad Gateway errors.
- **Data corruption or incomplete transactions**: A database write that was half-completed could leave records in an inconsistent state.
- **Connection leaks**: Active MongoDB client connection pool sockets remain open until server-side TCP timeouts expire.

### The Lifecycle of a Graceful Shutdown

A production-grade graceful shutdown reverses the startup process systematically:
1. **Stop accepting new incoming HTTP connections** while keeping active HTTP connections open.
2. **Wait for in-flight requests to complete** up to a maximum grace period (e.g., 10 seconds).
3. **Notify idle connections to close**.
4. **Close database connections** once all HTTP handlers have finished their database operations.
5. **Exit cleanly with exit code 0**.

```
[ OS Signal (SIGINT / SIGTERM) ]
               │
               ▼
[ 1. Catch signal via buffered channel ]
               │
               ▼
[ 2. Call srv.Shutdown(ctxWithTimeout) ]
       │──> Stop accepting new requests
       └──> Wait for active requests to finish
               │
               ▼
[ 3. Call app.Close(ctx) to close MongoDB ]
               │
               ▼
[ 4. Process exits cleanly (code 0) ]
```

---

## 2. Standard Library Primitives Used

Phase 8 uses only the Go standard library:

1. **`net/http` (`http.Server`)**:
   - Instead of calling `router.Run()` (which uses default infinite timeouts and blocks indefinitely), we configure an explicit `&http.Server{}` struct.
   - Provides fine-grained control over network timeouts (`ReadHeaderTimeout`, `IdleTimeout`) to prevent Slowloris and connection starvation attacks.
   - Provides `srv.Shutdown(ctx)`, which gracefully stops the server without dropping active requests.

2. **`os/signal` & `syscall`**:
   - `signal.Notify` registers a channel to receive specific OS signals.
   - `syscall.SIGINT` corresponds to `Ctrl + C` sent from a terminal.
   - `syscall.SIGTERM` is the standard termination signal sent by Kubernetes, Docker, and systemd to request graceful termination.

3. **`context.WithTimeout`**:
   - Establishes an upper time limit (deadline) for the shutdown procedure. If any slow HTTP request hangs longer than the grace period (e.g., 10 seconds), the context expires and forces the server to abort, preventing the process from hanging forever.

---

## 3. Step-by-Step Implementation Breakdown (`cmd/api/main.go`)

### Step 8.1: Package & Dependencies
- Package: `package main`
- Standard library imports:
  - `context` (for background and shutdown contexts)
  - `errors` (to check `http.ErrServerClosed`)
  - `log` (for structured startup and shutdown logging)
  - `net/http` (for `http.Server` and status handling)
  - `os` (for operating system signal notification)
  - `os/signal` (to hook into OS signals)
  - `syscall` (for `syscall.SIGINT`, `syscall.SIGTERM`)
  - `time` (for timeouts and durations)
- Project imports:
  - `blog-api/internal/app`

*(Note: `config`, `db`, `httpserver`, and domain packages are no longer directly needed in `main.go` because they are cleanly encapsulated inside `internal/app`!)*

---

### Step 8.2: Initialize Application Container
1. Create a root background context: `ctx := context.Background()`.
2. Initialize the application container:
   - Call `appInstance, err := app.NewApp(ctx)`.
   - If an error occurs, log fatal and terminate immediately (`log.Fatalf("Failed to initialize application: %v", err)`).
3. The `App` struct already contains the loaded `Config`, the wired `Router`, and the initialized `Database`.

---

### Step 8.3: Configure the Explicit `http.Server`
Define an `http.Server` pointer instead of calling `app.Router.Run()`:
- **`Addr`**: Bind address, formatted as `":" + appInstance.Config.Port` (or `fmt.Sprintf(":%s", appInstance.Config.Port)`).
- **`Handler`**: The Gin engine: `appInstance.Router`.
- **`ReadHeaderTimeout`**: Protection against Slowloris attacks (recommended: `5 * time.Second`).
- **`IdleTimeout`**: Maximum amount of time to wait for the next request when keep-alive is enabled (recommended: `60 * time.Second`).

---

### Step 8.4: Launch Server in a Background Goroutine
Because `srv.ListenAndServe()` blocks until the server stops, running it synchronously on the main goroutine would prevent you from listening for shutdown signals.

1. Launch a separate goroutine using `go func() { ... }()`.
2. Inside the goroutine, call:
   ```
   err := srv.ListenAndServe()
   ```
3. Check the returned error:
   - When `srv.Shutdown()` is triggered later, `ListenAndServe()` immediately returns `http.ErrServerClosed`.
   - **This is expected behavior, not a failure**.
   - Check with `if err != nil && !errors.Is(err, http.ErrServerClosed)`: only log a fatal error if the error is **not** `http.ErrServerClosed`.

---

### Step 8.5: Intercept OS Interrupt Signals
On the main goroutine, set up signal trapping so the process pauses until an interrupt occurs:

1. Create a buffered channel of `os.Signal` with capacity `1`:
   - `quit := make(chan os.Signal, 1)`
   - *(A buffer size of 1 is critical so the runtime does not drop signals if the channel is not read immediately).*
2. Register the channel to receive signals:
   - `signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)`
3. Block the main goroutine:
   - `<-quit`
   - Execution will halt at this line until `Ctrl + C` is pressed or the process receives a termination signal.
4. Log an informational message indicating shutdown has commenced:
   - `log.Println("Shutting down server...")`

---

### Step 8.6: Coordinated Graceful Teardown

Once the signal is received, execute the two-phase teardown:

#### Phase A: Drain and Stop the HTTP Server
1. Create a shutdown context with a 10-second deadline:
   - `shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)`
   - Always `defer cancel()` to release context resources.
2. Call `srv.Shutdown(shutdownCtx)`:
   - If `srv.Shutdown(shutdownCtx)` returns an error (e.g., deadline exceeded because a request took more than 10 seconds), log it:
     - `log.Printf("Server forced to shutdown: %v", err)`
   - Otherwise, log that the HTTP server exited gracefully.

#### Phase B: Clean Up Database Connections
1. **Only after** the HTTP server has finished in-flight requests, close the MongoDB connection pool:
   - Create another short timeout context (e.g., `5 * time.Second`) or reuse context:
     - `dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)`
     - `defer dbCancel()`
2. Call `appInstance.Close(dbCtx)`:
   - If closing fails, log the error.
   - Otherwise, log that the database connection was closed cleanly.
3. Log final message: `log.Println("Server exiting")`.

---

## 4. Key Gotchas & Best Practices

1. **Order of Shutdown is Non-Negotiable**:
   - **Never close the database before stopping the HTTP server**. If you close MongoDB while the HTTP server is still executing active requests, those active handlers will encounter sudden connection errors ("client is disconnected").
   - Always: `srv.Shutdown()` $\rightarrow$ then `app.Close()`.

2. **Always Check for `http.ErrServerClosed`**:
   - Calling `srv.Shutdown()` intentionally causes `srv.ListenAndServe()` to unblock and return `http.ErrServerClosed`. If you don't ignore this specific error, your application will log a false alarm fatal error on clean exits.

3. **Buffer the Signal Channel**:
   - Always use `make(chan os.Signal, 1)`. Unbuffered channels risk missing signals if the receiver isn't ready at the exact microsecond the signal arrives.

4. **Context Deadlines are Safety Nets**:
   - Always put a timeout on `srv.Shutdown`. Without a timeout, a rogue client that keeps an HTTP connection alive indefinitely could prevent your server process from ever shutting down.

---

## 5. Verification Checklist

After implementing `cmd/api/main.go`:
- [ ] Run `go build ./...` to verify there are no compilation errors.
- [ ] Start the server: `go run cmd/api/main.go`.
- [ ] Verify you see the server starting message and the port.
- [ ] In another terminal, ping `GET http://localhost:5000/health`.
- [ ] Press `Ctrl + C` in the server terminal:
  - [ ] Verify `Shutting down server...` is logged.
  - [ ] Verify MongoDB disconnects cleanly without errors.
  - [ ] Verify the process exits with code 0.
