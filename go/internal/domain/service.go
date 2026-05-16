package domain

import (
	"context"
	"fmt"

	"rebac-primer/internal/authz"
)

// DocumentOperations is the application-level interface for the documents use cases.
// HTTP handlers depend on this interface, not on the concrete service struct.
type DocumentOperations interface {
	Create(ctx context.Context, input CreateDocumentInput) (*CollaborativeDocument, error)
	Read(ctx context.Context, id string, actor authz.Object) (*CollaborativeDocument, error)
	Update(ctx context.Context, input UpdateDocumentInput) (*CollaborativeDocument, error)
}

// documentService wires a DocumentRepository to an Authorizer.
// Constructor injection keeps the fields unexported and prevents accidental mutation.
type documentService struct {
	repo DocumentRepository
	auth authz.Authorizer
}

// NewDocumentService creates a DocumentService with the given dependencies.
func NewDocumentService(repo DocumentRepository, auth authz.Authorizer) DocumentOperations {
	return &documentService{repo: repo, auth: auth}
}

// Create saves a new document if the actor has editor access to the workspace.
func (s *documentService) Create(ctx context.Context, input CreateDocumentInput) (*CollaborativeDocument, error) {
	if err := s.requireAllowed(ctx, input.Actor, authz.RelationWorkspaceEditor, input.Workspace, "create documents in"); err != nil {
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
	return &doc, nil
}

// Read returns a document if the actor has can_read access.
// It checks existence before authorization so the error message is accurate
// (a non-existent document should return not-found, not forbidden).
func (s *documentService) Read(ctx context.Context, id string, actor authz.Object) (*CollaborativeDocument, error) {
	doc, err := s.requireDocument(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.requireAllowed(ctx, actor, authz.RelationDocumentCanRead, authz.Document(id), "read"); err != nil {
		return nil, err
	}

	return doc, nil
}

// Update saves new body text if the actor has can_edit access.
func (s *documentService) Update(ctx context.Context, input UpdateDocumentInput) (*CollaborativeDocument, error) {
	existing, err := s.requireDocument(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if err := s.requireAllowed(ctx, input.Actor, authz.RelationDocumentCanEdit, authz.Document(input.ID), "edit"); err != nil {
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
	actor authz.Object,
	relation authz.Relation,
	object authz.Object,
	action string,
) error {
	result, err := s.auth.Check(ctx, authz.CheckRequest{
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
