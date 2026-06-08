package documents

import (
	"context"

	"rebac-primer/internal/rebac"
)

// Create saves a new document if the actor has editor access to the workspace.
//
// After persisting the document it writes two relationship tuples to the authz
// service so future checks can traverse the graph:
//
//	(document:id, workspace, workspace:X) — records where the document lives, so
//	                                        workspace members inherit access.
//	(document:id, owner,     user:actor)  — the creator directly owns the document
//	                                        (e.g. can_delete, an owner-only action).
//
// This is the write-back pattern: the documents service owns document-level
// tuples; the authz service owns workspace/team tuples.
func (s *documentService) Create(ctx context.Context, input CreateDocumentInput) (*CollaborativeDocument, error) {
	if err := s.requireAllowed(ctx,
		input.Actor,
		rebac.RelationWorkspaceEditor,
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
	if err := s.repo.Save(ctx, doc); err != nil {
		return nil, err
	}

	// Register the document relationships so the graph evaluator can resolve
	// can_read / can_edit for workspace members and owner-only actions for the
	// creator.
	if err := s.authzClient.WriteTuples(ctx, []rebac.TupleKey{
		rebac.Tuple(
			rebac.Document(input.ID),
			rebac.RelationDocumentWorkspace,
			rebac.Subject(input.Workspace),
		),
		rebac.Tuple(
			rebac.Document(input.ID),
			rebac.RelationDocumentOwner,
			rebac.Subject(input.Actor),
		),
	}); err != nil {
		return nil, err
	}

	return &doc, nil
}
