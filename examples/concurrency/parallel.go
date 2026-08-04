// Package concurrency is a Go-language teaching example, NOT part of the
// production ReBAC path. It demonstrates goroutines, channels, and WaitGroups by
// fanning out action checks. See docs/22-go-concurrency.md.
package concurrency

import (
	"context"
	"fmt"
	"sync"

	"rebac-primer/internal/rebac"
)

// Checker is the action-evaluation capability consumed by this example.
type Checker interface {
	Evaluate(ctx context.Context, req rebac.CheckRequest) (rebac.CheckResult, error)
}

// ActionSummary maps an Action to whether it is allowed for a subject
// and resource. It is the return type of [AllActions].
type ActionSummary map[rebac.Action]bool

// AllActions checks every action on a resource for a subject
// concurrently. It spawns one goroutine per action and collects results
// through a channel, returning when all checks complete or the context is done.
//
// Use this to build a "what can this subject do?" summary—for example, when a
// UI needs to know which action buttons to render.
func AllActions(ctx context.Context, auth Checker, subject rebac.Resource, resource rebac.Resource) (ActionSummary, error) {
	actions := actionsFor(resource)
	if len(actions) == 0 {
		return ActionSummary{}, nil
	}

	type outcome struct {
		action  rebac.Action
		allowed bool
		err     error
	}

	// Buffer the channel so goroutines never block if the receiver is slow.
	ch := make(chan outcome, len(actions))

	for _, action := range actions {
		go func(action rebac.Action) {
			result, err := auth.Evaluate(ctx, rebac.CheckRequest{
				Subject: subject, Action: action, Resource: resource,
			})
			ch <- outcome{action: action, allowed: result.Allowed, err: err}
		}(action)
	}

	summary := make(ActionSummary, len(actions))
	for range len(actions) {
		// select waits on whichever happens first: the next result arriving, or
		// the caller's context being cancelled / timing out.
		select {
		case out := <-ch:
			if out.err != nil {
				return nil, fmt.Errorf("check %s: %w", out.action, out.err)
			}
			summary[out.action] = out.allowed
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
// returns results in the same order as the input slice.  Unlike AllActions,
// it works with arbitrary (subject, action, resource) combinations.
//
// If any check returns an error the corresponding Err field is set; the other
// results are still returned.  The caller decides whether to treat any error as
// fatal.
func BulkCheck(ctx context.Context, auth Checker, reqs []rebac.CheckRequest) []BulkResult {
	results := make([]BulkResult, len(reqs))
	var wg sync.WaitGroup

	for i, req := range reqs {
		wg.Go(func() {
			result, err := auth.Evaluate(ctx, req)
			results[i] = BulkResult{Request: req, Result: result, Err: err}
		})
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

// actionsFor returns the actions that make sense for a resource type.
func actionsFor(resource rebac.Resource) []rebac.Action {
	typ, _, err := rebac.ParseResource(string(resource))
	if err != nil {
		return nil
	}
	if typ == rebac.ResourceTypeDocument {
		return []rebac.Action{
			rebac.ActionDocumentRead,
			rebac.ActionDocumentComment,
			rebac.ActionDocumentEdit,
			rebac.ActionDocumentDelete,
		}
	}
	return nil
}
