package domain

import (
	"context"

	"rebac-primer/internal/shared"
)

// Read returns a document if the actor has can_read access.
//
// It checks existence before authorization so the error message is accurate:
// a non-existent document returns not-found, not forbidden.
//
// Mirrors typescript/src/documents-service/core/domain/makeReadDocument.ts.
func (s *documentService) Read(ctx context.Context, id string, actor shared.Object) (*CollaborativeDocument, error) {
	doc, err := s.requireDocument(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.requireAllowed(ctx,
		actor,
		shared.RelationDocumentCanRead,
		shared.Document(id),
		"read",
	); err != nil {
		return nil, err
	}

	return doc, nil
}
