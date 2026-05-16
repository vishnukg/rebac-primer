package domain

import (
	"context"
	"sync"
)

// DocumentRepository is the persistence interface for CollaborativeDocument.
// The domain service depends only on this interface — it never knows whether
// documents are stored in memory, Postgres, or somewhere else.
type DocumentRepository interface {
	Save(ctx context.Context, doc CollaborativeDocument) error
	FindByID(ctx context.Context, id string) (*CollaborativeDocument, error)
	List(ctx context.Context) ([]CollaborativeDocument, error)
}

// InMemoryDocumentRepository is a map-backed DocumentRepository for tests and demos.
type InMemoryDocumentRepository struct {
	mu   sync.RWMutex
	docs map[string]CollaborativeDocument
}

// NewInMemoryDocumentRepository creates an empty repository.
func NewInMemoryDocumentRepository() *InMemoryDocumentRepository {
	return &InMemoryDocumentRepository{
		docs: make(map[string]CollaborativeDocument),
	}
}

// Save stores or replaces a document. It is idempotent on the document ID.
func (r *InMemoryDocumentRepository) Save(_ context.Context, doc CollaborativeDocument) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.docs[doc.ID] = doc
	return nil
}

// FindByID retrieves a document by ID. Returns nil (not an error) when not found.
func (r *InMemoryDocumentRepository) FindByID(_ context.Context, id string) (*CollaborativeDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	doc, ok := r.docs[id]
	if !ok {
		return nil, nil
	}
	return &doc, nil
}

// List returns all documents in arbitrary order.
func (r *InMemoryDocumentRepository) List(_ context.Context) ([]CollaborativeDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]CollaborativeDocument, 0, len(r.docs))
	for _, doc := range r.docs {
		out = append(out, doc)
	}
	return out, nil
}
