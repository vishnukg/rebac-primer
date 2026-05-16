package authz

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"
)

// AuditAuthorizer wraps any Authorizer and writes a one-line audit record for
// every Check call. It is a decorator: it adds behaviour without touching the
// inner implementation.
//
// This is the classic Go middleware pattern: take an interface, return the same
// interface, do something before/after the inner call.
type AuditAuthorizer struct {
	inner  Authorizer
	logger *log.Logger
}

// NewAuditAuthorizer wraps inner with audit logging.
// Pass io.Discard as w to silence output in tests.
func NewAuditAuthorizer(inner Authorizer, w io.Writer) *AuditAuthorizer {
	return &AuditAuthorizer{
		inner:  inner,
		logger: log.New(w, "[authz] ", 0),
	}
}

// Check delegates to the inner Authorizer and then logs the outcome.
// The AuditAuthorizer satisfies the Authorizer interface, so it can be dropped
// in anywhere an Authorizer is expected without changing call sites.
func (a *AuditAuthorizer) Check(ctx context.Context, req CheckRequest) (CheckResult, error) {
	start := time.Now()
	result, err := a.inner.Check(ctx, req)
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

// Compile-time assertion: *AuditAuthorizer must implement Authorizer.
// If the method set ever drifts, the compiler catches it here rather than at
// the call site where the nil value is assigned.
var _ Authorizer = (*AuditAuthorizer)(nil)

// ReadOnlyStore embeds TupleReader to expose only read operations from an
// InMemoryTupleStore. Embedding promotes all TupleReader methods onto
// ReadOnlyStore so callers get Has and FindByObjectRelation without any glue
// code.
//
// The write methods (Write, Delete) are not promoted because TupleWriter is not
// embedded — they are simply absent. The compiler enforces the restriction.
type ReadOnlyStore struct {
	TupleReader
}

// NewReadOnlyStore wraps a store so it can be passed to code that must not
// write tuples — for example, a read-only replica or a test spy.
func NewReadOnlyStore(r TupleReader) ReadOnlyStore {
	return ReadOnlyStore{r}
}
