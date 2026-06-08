package documents

import (
	"context"

	"rebac-primer/internal/rebac"
)

// Read returns a document if the actor has can_read access.
//
// Existence is checked before authorization so the error is accurate:
// a non-existent document returns not-found, not forbidden.
//
// Security tradeoff: this ordering leaks existence. A denied actor gets 403 for a
// document that exists but 404 for one that does not, so they can probe which ids
// exist even without access. That is fine for this tutorial — clear errors aid
// learning — but high-security systems return 404 for both cases so the two are
// indistinguishable (check authorization first, then map a denial to not-found).
// See docs/40-production-readiness.md (Gap 13).
func (s *documentService) Read(ctx context.Context, id string, actor rebac.Object) (*CollaborativeDocument, error) {
	doc, err := s.requireDocument(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.requireAllowed(ctx,
		actor,
		rebac.RelationDocumentCanRead,
		rebac.Document(id),
		"read",
	); err != nil {
		return nil, err
	}

	return doc, nil
}
