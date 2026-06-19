package documents

import (
	"context"
	"sync"
)

// InMemoryRepository is a map-backed [DocumentRepository] for tests and demos.
// Save stores a struct copy, so mutating the caller's value after Save does not
// affect the stored document (snapshot semantics).
type InMemoryRepository struct {
	mu   sync.RWMutex
	docs map[string]CollaborativeDocument
}

// NewInMemoryRepository creates an empty repository.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		docs: make(map[string]CollaborativeDocument),
	}
}

// Compile-time assertion: *InMemoryRepository must satisfy DocumentRepository.
var _ DocumentRepository = (*InMemoryRepository)(nil)

// Create stores a new document and atomically rejects an existing ID.
func (r *InMemoryRepository) Create(_ context.Context, doc CollaborativeDocument) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.docs[doc.ID]; exists {
		return &DocumentAlreadyExistsError{ID: doc.ID}
	}
	r.docs[doc.ID] = doc
	return nil
}

// Save stores or replaces a document (idempotent on ID).
func (r *InMemoryRepository) Save(_ context.Context, doc CollaborativeDocument) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.docs[doc.ID] = doc // struct copy → snapshot semantics
	return nil
}

// FindByID retrieves a document by ID. Returns nil (not an error) when not found.
func (r *InMemoryRepository) FindByID(_ context.Context, id string) (*CollaborativeDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	doc, ok := r.docs[id]
	if !ok {
		return nil, nil
	}
	return &doc, nil
}

// Delete removes a document. It is a no-op when the ID does not exist.
func (r *InMemoryRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.docs, id)
	return nil
}
