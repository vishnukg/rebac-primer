package documents

import (
	"context"
	"errors"
	"fmt"
	"time"

	"rebac-primer/internal/rebac"
)

// Service creates, reads, and updates collaborative documents.
// Construct it with [New]; its zero value is not usable.
type Service struct {
	repo  DocumentRepository
	authz AuthorizationService
}

// New creates a Service from a DocumentRepository and an AuthorizationService.
func New(repo DocumentRepository, authz AuthorizationService) *Service {
	return &Service{repo: repo, authz: authz}
}

// ── Operations ────────────────────────────────────────────────────────────────

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
func (s *Service) Create(ctx context.Context, input CreateDocumentInput) (*CollaborativeDocument, error) {
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
	// Create is atomic at the repository boundary. A separate FindByID followed
	// by Save would allow concurrent requests to race and overwrite the same ID.
	if err := s.repo.Create(ctx, doc); err != nil {
		return nil, err
	}

	// Register the document relationships so the graph evaluator can resolve
	// can_read / can_edit for workspace members and owner-only actions for the
	// creator.
	tuples := []rebac.TupleKey{
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
	}
	if err := s.authz.WriteTuples(ctx, tuples); err != nil {
		// The document and authorization stores do not share a transaction.
		// Compensate on failure so the demo does not leave an inaccessible
		// document or partially written relationships behind. Cleanup gets a
		// short independent deadline because the request may already be canceled.
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		cleanupErr := errors.Join(
			s.authz.DeleteTuples(cleanupCtx, tuples),
			s.repo.Delete(cleanupCtx, input.ID),
		)
		if cleanupErr != nil {
			return nil, errors.Join(err, fmt.Errorf("rollback failed: %w", cleanupErr))
		}
		return nil, err
	}

	return &doc, nil
}

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
// See the Security Notes in docs/40-production-readiness.md.
func (s *Service) Read(ctx context.Context, id string, actor rebac.Object) (*CollaborativeDocument, error) {
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

// Update saves new body text if the actor has can_edit access.
func (s *Service) Update(ctx context.Context, input UpdateDocumentInput) (*CollaborativeDocument, error) {
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

// ── Shared helpers ──────────────────────────────────────────────────────────────

// requireDocument fetches a document and wraps a missing one in [DocumentNotFoundError].
func (s *Service) requireDocument(ctx context.Context, id string) (*CollaborativeDocument, error) {
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
func (s *Service) requireAllowed(
	ctx context.Context,
	actor rebac.Object,
	relation rebac.Relation,
	object rebac.Object,
	action string,
) error {
	result, err := s.authz.Check(ctx, rebac.CheckRequest{
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
