package domain

import (
	"context"

	"rebac-primer/internal/shared"
)

// Create saves a new document if the actor has editor access to the workspace.
//
// After persisting the document it writes a "document belongs to workspace"
// tuple so subsequent can_read checks can traverse the graph.
//
// Mirrors typescript/src/documents-service/core/domain/makeCreateDocument.ts.
func (s *documentService) Create(ctx context.Context, input CreateDocumentInput) (*CollaborativeDocument, error) {
	if err := s.requireAllowed(ctx,
		input.Actor,
		shared.RelationWorkspaceEditor,
		input.Workspace,
		"create documents in",
	); err != nil {
		return nil, err
	}

	doc := CollaborativeDocument{
		ID:        input.ID,
		Title:     input.Title,
		Body:      input.Body,
		Workspace: input.Workspace,
		UpdatedBy: input.Actor,
	}
	if err := s.repo.Save(ctx, doc.toPort()); err != nil {
		return nil, err
	}

	// Register the document → workspace relationship so the graph evaluator can
	// resolve can_read / can_edit for workspace members.
	if err := s.authzClient.WriteTuples(ctx, []shared.TupleKey{
		shared.Tuple(
			shared.Document(input.ID),
			shared.RelationDocumentWorkspace,
			shared.Subject(input.Workspace),
		),
	}); err != nil {
		return nil, err
	}

	return &doc, nil
}
