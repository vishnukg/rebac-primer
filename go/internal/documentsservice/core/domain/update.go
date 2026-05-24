package domain

import (
	"context"

	"rebac-primer/internal/shared"
)

// Update saves new body text if the actor has can_edit access.
//
// Mirrors typescript/src/documents-service/core/domain/makeUpdateDocument.ts.
func (s *documentService) Update(ctx context.Context, input UpdateDocumentInput) (*CollaborativeDocument, error) {
	existing, err := s.requireDocument(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if err := s.requireAllowed(ctx,
		input.Actor,
		shared.RelationDocumentCanEdit,
		shared.Document(input.ID),
		"edit",
	); err != nil {
		return nil, err
	}

	updated := *existing
	updated.Body = input.Body
	updated.UpdatedBy = input.Actor

	if err := s.repo.Save(ctx, updated); err != nil {
		return nil, err
	}
	return &updated, nil
}
