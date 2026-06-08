package documents

import (
	"context"
	"fmt"

	"rebac-primer/internal/rebac"
)

// documentService is the concrete implementation returned by [New].
// It is unexported — callers hold a [Service] interface value.
type documentService struct {
	repo        DocumentRepository
	authzClient AuthzClient
}

// New creates a [Service] from a DocumentRepository and an AuthzClient.
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
	actor rebac.Object,
	relation rebac.Relation,
	object rebac.Object,
	action string,
) error {
	result, err := s.authzClient.Check(ctx, rebac.CheckRequest{
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
