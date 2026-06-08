package documents

import (
	"context"

	"rebac-primer/internal/rebac"
)

// Update saves new body text if the actor has can_edit access.
func (s *documentService) Update(ctx context.Context, input UpdateDocumentInput) (*CollaborativeDocument, error) {
	existing, err := s.requireDocument(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if err := s.requireAllowed(ctx,
		input.Actor,
		rebac.RelationDocumentCanEdit,
		rebac.Document(input.ID),
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
