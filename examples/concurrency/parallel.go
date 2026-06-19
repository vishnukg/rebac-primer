// Package concurrency is a Go-language teaching example, NOT part of the
// production ReBAC path. It demonstrates goroutines, channels, and WaitGroups by
// fanning out permission checks. See docs/22-go-concurrency.md.
package concurrency

import (
	"context"
	"fmt"
	"sync"

	"rebac-primer/internal/rebac"
)

// Checker is the permission-evaluation capability consumed by this example.
type Checker interface {
	Evaluate(ctx context.Context, req rebac.CheckRequest) (rebac.CheckResult, error)
}

// PermissionSummary maps a Relation to whether it is allowed for a given user
// and object.  It is the return type of [AllPermissions].
type PermissionSummary map[rebac.Relation]bool

// AllPermissions checks every computed permission on an object for a user
// concurrently.  It spawns one goroutine per relation and collects results
// through a channel, returning when all checks complete or the context is done.
//
// Use this to build a "what can this user do?" summary — for example, when a
// UI needs to know which action buttons to render.
func AllPermissions(ctx context.Context, auth Checker, user rebac.Object, object rebac.Object) (PermissionSummary, error) {
	relations := computedRelationsFor(object)
	if len(relations) == 0 {
		return PermissionSummary{}, nil
	}

	type outcome struct {
		relation rebac.Relation
		allowed  bool
		err      error
	}

	// Buffer the channel so goroutines never block if the receiver is slow.
	ch := make(chan outcome, len(relations))

	for _, rel := range relations {
		go func(rel rebac.Relation) {
			result, err := auth.Evaluate(ctx, rebac.CheckRequest{User: user, Relation: rel, Object: object})
			ch <- outcome{relation: rel, allowed: result.Allowed, err: err}
		}(rel)
	}

	summary := make(PermissionSummary, len(relations))
	for range len(relations) {
		// select waits on whichever happens first: the next result arriving, or
		// the caller's context being cancelled / timing out.
		select {
		case out := <-ch:
			if out.err != nil {
				return nil, fmt.Errorf("check %s: %w", out.relation, out.err)
			}
			summary[out.relation] = out.allowed
		case <-ctx.Done():
			// Caller cancelled or timed out. Return its reason immediately.
			// The still-running goroutines each send one value into ch, which is
			// buffered with room for every result, so they finish and exit
			// without blocking — no goroutine leak even though we stopped early.
			return nil, ctx.Err()
		}
	}

	return summary, nil
}

// BulkCheck runs a list of CheckRequests concurrently using a WaitGroup and
// returns results in the same order as the input slice.  Unlike AllPermissions,
// it works with arbitrary (user, relation, object) combinations.
//
// If any check returns an error the corresponding Err field is set; the other
// results are still returned.  The caller decides whether to treat any error as
// fatal.
func BulkCheck(ctx context.Context, auth Checker, reqs []rebac.CheckRequest) []BulkResult {
	results := make([]BulkResult, len(reqs))
	var wg sync.WaitGroup

	for i, req := range reqs {
		wg.Add(1)
		go func(i int, req rebac.CheckRequest) {
			defer wg.Done()
			result, err := auth.Evaluate(ctx, req)
			results[i] = BulkResult{Request: req, Result: result, Err: err}
		}(i, req)
	}

	wg.Wait()
	return results
}

// BulkResult holds the outcome of one check from a [BulkCheck] call.
type BulkResult struct {
	Request rebac.CheckRequest
	Result  rebac.CheckResult
	Err     error
}

// computedRelationsFor returns the computed (action) relations that make sense
// to check for a given object type.
func computedRelationsFor(object rebac.Object) []rebac.Relation {
	typ, _, err := rebac.ParseObject(string(object))
	if err != nil {
		return nil
	}
	if typ == rebac.ObjectTypeDocument {
		return []rebac.Relation{
			rebac.RelationDocumentCanRead,
			rebac.RelationDocumentCanComment,
			rebac.RelationDocumentCanEdit,
			rebac.RelationDocumentCanDelete,
		}
	}
	return nil
}
