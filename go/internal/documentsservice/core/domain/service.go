package domain

import (
	"context"
	"fmt"

	"rebac-primer/internal/documentsservice/core/ports"
	"rebac-primer/internal/shared"
)

// Documents is the driving port — what the HTTP handler and tests depend on.
// Mirrors typescript/src/documents-service/core/domain/types.ts (Documents type).
type Documents interface {
	Create(ctx context.Context, input CreateDocumentInput) (*CollaborativeDocument, error)
	Read(ctx context.Context, id string, actor shared.Object) (*CollaborativeDocument, error)
	Update(ctx context.Context, input UpdateDocumentInput) (*CollaborativeDocument, error)
}

// documentService wires the two driven ports together and exposes Documents.
type documentService struct {
	repo        ports.DocumentRepository
	authzClient ports.AuthzClient
}

// New creates a Documents service from its two driven ports.
// This is the Go equivalent of makeDocuments() in TypeScript.
func New(repo ports.DocumentRepository, authzClient ports.AuthzClient) Documents {
	return &documentService{repo: repo, authzClient: authzClient}
}

// ── Shared helpers ────────────────────────────────────────────────────────────

// requireDocument fetches a document and wraps a missing one in DocumentNotFoundError.
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

// requireAllowed runs an authorization check and returns ForbiddenError on denial.
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
