package graph

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"rebac-primer/internal/authzservice/core/ports"
	"rebac-primer/internal/shared"
)

// AuditEvaluator wraps any Checker (ports.Evaluator) and writes a one-line
// audit record for every Evaluate call. It is a decorator: it adds behaviour
// without touching the inner implementation.
//
// This is the classic Go middleware pattern: take an interface, return the same
// interface, do something before/after the inner call.
type AuditEvaluator struct {
	inner  Checker // Checker = ports.Evaluator
	logger *log.Logger
}

// NewAuditEvaluator wraps inner with audit logging.
// Pass io.Discard as w to silence output in tests.
func NewAuditEvaluator(inner Checker, w io.Writer) *AuditEvaluator {
	return &AuditEvaluator{
		inner:  inner,
		logger: log.New(w, "[authz] ", 0),
	}
}

// Evaluate delegates to the inner Checker and then logs the outcome.
// AuditEvaluator satisfies the Checker interface, so it can be dropped in
// anywhere a Checker is expected without changing call sites.
func (a *AuditEvaluator) Evaluate(ctx context.Context, req shared.CheckRequest) (shared.CheckResult, error) {
	start := time.Now()
	result, err := a.inner.Evaluate(ctx, req)
	elapsed := time.Since(start)

	status := "allowed"
	if err != nil {
		status = fmt.Sprintf("error: %v", err)
	} else if !result.Allowed {
		status = "denied"
	}

	a.logger.Printf("check user=%s relation=%s object=%s -> %s (%s)",
		req.User, req.Relation, req.Object, status, elapsed)

	return result, err
}

// Compile-time assertion: *AuditEvaluator must implement Checker (= ports.Evaluator).
var _ Checker = (*AuditEvaluator)(nil)

// ── ReadOnlyStore ─────────────────────────────────────────────────────────────

// ReadOnlyStore embeds ports.TupleRepository's read methods to expose only the
// read operations from an InMemoryTupleStore.
//
// Embedding promotes all TupleRepository read methods onto ReadOnlyStore, but
// the write methods (Write, Delete) are absent — the compiler enforces that.
type ReadOnlyStore struct {
	ports.TupleRepository
}

// NewReadOnlyStore wraps a repository so it can be passed to code that must not
// write tuples — for example, a read-only replica or a test spy.
func NewReadOnlyStore(r ports.TupleRepository) ReadOnlyStore {
	return ReadOnlyStore{r}
}
