# Go HTTP, JSON, Context, and Application Lifecycle

The Go standard library is sufficient to build a production-shaped HTTP
service. This chapter explains the APIs used by this repository before the
request-flow walkthrough applies them.

## `net/http` Mental Model

An HTTP handler implements:

```go
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}
```

The function adapter `http.HandlerFunc` lets an ordinary function satisfy that
interface:

```go
func health(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusNoContent)
}
```

Go's method-aware router patterns can include path variables:

```go
mux := http.NewServeMux()
mux.HandleFunc("GET /documents/{id}", handleGetDocument)
```

Retrieve the variable with:

```go
id := r.PathValue("id")
```

Handlers may run concurrently. Shared mutable state must be synchronized or
confined to a request.

## Requests and Responses

The request contains method, URL, headers, body, and context:

```go
authorization := r.Header.Get("Authorization")
ctx := r.Context()
defer r.Body.Close()
```

Set response headers before writing the status or body:

```go
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusCreated)
```

The first body write implicitly sends status 200 if no status was written.
After headers are sent, changing the status has no effect.

Use named status constants such as `http.StatusNotFound` rather than numeric
literals.

## JSON

Struct tags control JSON field names:

```go
type updateRequest struct {
    Body string `json:"body"`
}
```

Decode from a stream:

```go
var input updateRequest
decoder := json.NewDecoder(r.Body)
decoder.DisallowUnknownFields()
if err := decoder.Decode(&input); err != nil {
    // map malformed input to HTTP 400
}
```

Encode a response:

```go
if err := json.NewEncoder(w).Encode(response); err != nil {
    // the response may already be partially written
}
```

Passing `&input` lets the decoder mutate the struct.

For untrusted requests, limit body size before decoding:

```go
r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
```

Deciding whether to reject unknown fields, trailing JSON values, and unsupported
media types is part of the HTTP boundary's contract.

## Context

`context.Context` carries cancellation, deadlines, and request-scoped metadata
across API boundaries:

```go
func (s *Service) Read(ctx context.Context, id string) error
```

Place it first and do not store it in a long-lived struct.

Create a deadline:

```go
ctx, cancel := context.WithTimeout(parent, 2*time.Second)
defer cancel()
```

Check cancellation:

```go
if err := ctx.Err(); err != nil {
    return err
}
```

Wait for cancellation alongside other work:

```go
select {
case result := <-results:
    return result
case <-ctx.Done():
    return ctx.Err()
}
```

Request contexts are canceled when the client connection closes, the server
cancels the request, or a parent deadline expires. Pass the same context into
database and HTTP client calls so abandoned work can stop.

Context values should contain request-scoped metadata crossing API boundaries,
not optional function parameters or general application dependencies.

## Synchronization

HTTP handlers can call in-memory repositories concurrently. A mutex protects an
invariant involving shared memory:

```go
type Store struct {
    mu    sync.RWMutex
    items map[string]Item
}

func (s *Store) Find(id string) (Item, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    item, ok := s.items[id]
    return item, ok
}
```

`RWMutex` permits multiple readers or one writer. A plain `Mutex` is often the
better default unless read concurrency has measured value.

Do not hold a mutex while performing slow network I/O or calling unknown code.

## Starting and Stopping a Server

A configured server provides timeouts:

```go
server := &http.Server{
    Addr:              ":4001",
    Handler:           mux,
    ReadHeaderTimeout: 5 * time.Second,
    ReadTimeout:       15 * time.Second,
    WriteTimeout:      30 * time.Second,
    IdleTimeout:       60 * time.Second,
}
```

`ListenAndServe` blocks, so a process that also waits for operating-system
signals normally starts it in a goroutine:

```go
errCh := make(chan error, 1)
go func() {
    errCh <- server.ListenAndServe()
}()
```

Graceful shutdown stops accepting new requests and waits for active handlers:

```go
shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
if err := server.Shutdown(shutdownCtx); err != nil {
    return err
}
```

`http.ErrServerClosed` is the expected result when shutdown closes the server.

## HTTP Tests

Use `httptest` without opening a real network port:

```go
request := httptest.NewRequest(http.MethodGet, "/health", nil)
response := httptest.NewRecorder()

handler.ServeHTTP(response, request)

if response.Code != http.StatusNoContent {
    t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
}
```

For integration behavior requiring a real HTTP client, use
`httptest.NewServer`. Close it with `defer server.Close()`.

Test status, important headers, and decoded response bodies. Also verify that
malformed input is rejected before domain dependencies are called.

## Process Configuration

Read environment variables at the composition root:

```go
port := os.Getenv("PORT")
```

Parse and validate configuration before starting the server. Avoid reading
environment variables throughout domain packages because it hides dependencies
and complicates tests.

Use `log.Fatal` only at the process boundary: it exits immediately and therefore
does not run deferred calls. Library and domain code should return errors.

## Try It

Run:

```bash
go test -v ./internal/api
go test -v ./cmd/server
go run ./cmd/server
```

In another terminal:

```bash
curl -i http://127.0.0.1:4001/health
curl -i http://127.0.0.1:4001/documents/roadmapDocument \
  -H "Authorization: Bearer demo-token-bob"
```

Then trace the second request from `internal/api/server.go` through
`internal/api/handler.go`, `internal/documents/service.go`, and
`internal/authz/service.go`.

## Checkpoint

You are ready to continue when you can explain:

- why handlers must assume concurrent execution
- why JSON decoding receives a pointer
- how request cancellation reaches a repository or HTTP client
- why response headers must be set before the status
- why configuration and shutdown belong in `cmd/server`

Next: [Go idioms and patterns](14-go-idioms-and-patterns.md).
