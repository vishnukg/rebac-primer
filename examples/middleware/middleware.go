// Package middleware is a Go-language teaching example, NOT part of the
// production ReBAC path. It demonstrates the decorator pattern (AuditEvaluator)
// and interface embedding (ReadOnlyStore). See docs/24-go-interfaces-embedding.md.
package middleware

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/rebac"
)

// Checker is the permission-evaluation capability consumed by this decorator.
type Checker interface {
	Evaluate(ctx context.Context, req rebac.CheckRequest) (rebac.CheckResult, error)
}

// AuditEvaluator wraps any [Checker] and writes a one-line
// audit record for every Evaluate call.  It is a decorator: it adds behaviour
// without touching the inner implementation.
//
// This is the classic Go middleware pattern: take an interface, return the same
// interface, do something before/after the inner call.
type AuditEvaluator struct {
	inner  Checker
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
func (a *AuditEvaluator) Evaluate(ctx context.Context, req rebac.CheckRequest) (rebac.CheckResult, error) {
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

// Compile-time assertion: *AuditEvaluator must implement Checker.
var _ Checker = (*AuditEvaluator)(nil)

// ── ReadOnlyStore ─────────────────────────────────────────────────────────────

// ReadOnlyStore embeds [authz.TupleReader]. Embedding an interface promotes the
// reader methods onto the outer struct, but not the write methods. This makes
// read-only intent a compiler-checked capability: callers that only receive a
// ReadOnlyStore cannot call Write or Delete through that value.
// See docs/24-go-interfaces-embedding.md.
type ReadOnlyStore struct {
	authz.TupleReader
}

// NewReadOnlyStore wraps a repository so it can be passed to code that should
// only read tuples — for example, a read-only replica or a test spy.
func NewReadOnlyStore(r authz.TupleReader) ReadOnlyStore {
	return ReadOnlyStore{r}
}
