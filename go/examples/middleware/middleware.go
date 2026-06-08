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
	"rebac-primer/internal/shared"
)

// Checker is a type alias for [authz.Evaluator] — see the concurrency example
// for the same alias. It lets the decorator below name the thing it wraps.
type Checker = authz.Evaluator

// AuditEvaluator wraps any [Checker] ([authz.Evaluator]) and writes a one-line
// audit record for every Evaluate call.  It is a decorator: it adds behaviour
// without touching the inner implementation.
//
// This is the classic Go middleware pattern: take an interface, return the same
// interface, do something before/after the inner call.
type AuditEvaluator struct {
	inner  Checker // Checker = authz.Evaluator
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

// Compile-time assertion: *AuditEvaluator must implement Checker (= authz.Evaluator).
var _ Checker = (*AuditEvaluator)(nil)

// ── ReadOnlyStore ─────────────────────────────────────────────────────────────

// ReadOnlyStore embeds [authz.TupleRepository]. Embedding an interface promotes
// every method of that interface onto the outer struct — including Write and
// Delete. So ReadOnlyStore is NOT read-only at the type level: a caller could
// still invoke ro.Write(...) or ro.Delete(...) and it would compile and mutate
// the underlying store.
//
// The name therefore expresses intent, not a compiler-enforced guarantee: pass
// a ReadOnlyStore to code that should only read tuples and let naming + review
// communicate that. To make read-only a guarantee the compiler checks, embed a
// narrower interface that omits Write and Delete (e.g. a reader interface with
// just Has/FindByObjectRelation/FindAll) instead of the full TupleRepository.
// See docs/24-go-interfaces-embedding.md.
type ReadOnlyStore struct {
	authz.TupleRepository
}

// NewReadOnlyStore wraps a repository so it can be passed to code that should
// only read tuples — for example, a read-only replica or a test spy. See the
// caveat on [ReadOnlyStore]: this signals intent rather than enforcing it.
func NewReadOnlyStore(r authz.TupleRepository) ReadOnlyStore {
	return ReadOnlyStore{r}
}
