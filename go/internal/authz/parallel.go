package authz

import (
	"context"
	"fmt"
	"sync"
)

// PermissionSummary maps a Relation to whether it is allowed for a given user
// and object. It is the return type of AllPermissions.
type PermissionSummary map[Relation]bool

// AllPermissions checks every computed permission on an object for a user
// concurrently. It spawns one goroutine per relation and collects results
// through a channel, returning when all checks complete or the context is done.
//
// Use this to build a "what can this user do?" summary — for example, when a
// UI needs to know which action buttons to render.
func AllPermissions(ctx context.Context, auth Authorizer, user Object, object Object) (PermissionSummary, error) {
	relations := computedRelationsFor(object)
	if len(relations) == 0 {
		return PermissionSummary{}, nil
	}

	type outcome struct {
		relation Relation
		allowed  bool
		err      error
	}

	// Buffer the channel so goroutines never block if the receiver is slow.
	ch := make(chan outcome, len(relations))

	for _, rel := range relations {
		go func(rel Relation) {
			result, err := auth.Check(ctx, CheckRequest{User: user, Relation: rel, Object: object})
			ch <- outcome{relation: rel, allowed: result.Allowed, err: err}
		}(rel)
	}

	summary := make(PermissionSummary, len(relations))
	for range len(relations) {
		out := <-ch
		if out.err != nil {
			return nil, fmt.Errorf("check %s: %w", out.relation, out.err)
		}
		summary[out.relation] = out.allowed
	}

	return summary, nil
}

// BulkCheck runs a list of CheckRequests concurrently using a WaitGroup and
// returns results in the same order as the input slice. Unlike AllPermissions,
// it works with arbitrary (user, relation, object) combinations.
//
// If any check returns an error the corresponding Err field is set; the other
// results are still returned. The caller decides whether to treat any error as
// fatal.
func BulkCheck(ctx context.Context, auth Authorizer, reqs []CheckRequest) []BulkResult {
	results := make([]BulkResult, len(reqs))
	var wg sync.WaitGroup

	for i, req := range reqs {
		wg.Add(1)
		// Capture i and req — the loop variables are re-used each iteration, so
		// passing them as arguments gives each goroutine its own copy.
		go func(i int, req CheckRequest) {
			defer wg.Done()
			result, err := auth.Check(ctx, req)
			results[i] = BulkResult{Request: req, Result: result, Err: err}
		}(i, req)
	}

	wg.Wait()
	return results
}

// BulkResult holds the outcome of one check from a BulkCheck call.
type BulkResult struct {
	Request CheckRequest
	Result  CheckResult
	Err     error
}

// computedRelationsFor returns the computed (action) relations that make sense
// to check for a given object. Only document-type objects have computed
// permissions; other types expose base relations only.
func computedRelationsFor(object Object) []Relation {
	typ, _, err := ParseObject(string(object))
	if err != nil {
		return nil
	}
	if typ == ObjectTypeDocument {
		return []Relation{
			RelationDocumentCanRead,
			RelationDocumentCanComment,
			RelationDocumentCanEdit,
			RelationDocumentCanDelete,
		}
	}
	return nil
}
