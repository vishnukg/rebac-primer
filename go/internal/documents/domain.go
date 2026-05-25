package documents

import (
	"context"
	"fmt"

	"rebac-primer/internal/shared"
)

// documentService wires the two driven ports and exposes [Service].
// It is unexported — callers hold a [Service] interface value.
type documentService struct {
	repo        DocumentRepository
	authzClient AuthzClient
}

// New creates a [Service] from its two driven ports.
// This is the Go equivalent of makeDocuments() in TypeScript.
//
// Mirrors typescript/src/documents-service/core/domain/makeDocuments.ts.
func New(repo DocumentRepository, authzClient AuthzClient) Service {
	return &documentService{repo: repo, authzClient: authzClient}
}

// ── Shared helpers ────────────────────────────────────────────────────────────

// requireDocument fetches a document and wraps a missing one in [DocumentNotFoundError].
func (s *documentService) requireDocument(ctx context.Context, id string) (*CollaborativeDocument, error) {
	doc, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, &DocumentNotFoundError{ID: id}
	}
	return doc, nil
}

// requireAllowed runs an authorization check and returns [ForbiddenError] on denial.
func (s *documentService) requireAllowed(
	ctx context.Context,
	actor shared.Object,
	relation shared.Relation,
	object shared.Object,
	action string,
) error {
	result, err := s.authzClient.Check(ctx, shared.CheckRequest{
		User:     actor,
		Relation: relation,
		Object:   object,
	})
	if err != nil {
		return err
	}
	if !result.Allowed {
		return &ForbiddenError{Message: fmt.Sprintf("%s cannot %s %s", actor, action, object)}
	}
	return nil
}
