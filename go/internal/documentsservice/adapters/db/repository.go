// Package db provides the in-memory DocumentRepository adapter.
//
// Mirrors typescript/src/documents-service/adapters/db/makeInMemoryDocumentRepository.ts.
package db

import (
	"context"
	"sync"

	"rebac-primer/internal/documentsservice/core/ports"
)

// InMemoryDocumentRepository is a map-backed DocumentRepository for tests and demos.
// Save stores a struct copy — mutations to the caller's value after Save do not
// affect the stored document (snapshot semantics, just like the TypeScript version).
type InMemoryDocumentRepository struct {
	mu   sync.RWMutex
	docs map[string]ports.CollaborativeDocument
}

// New creates an empty repository.
func New() *InMemoryDocumentRepository {
	return &InMemoryDocumentRepository{
		docs: make(map[string]ports.CollaborativeDocument),
	}
}

// Save stores or replaces a document (idempotent on ID).
func (r *InMemoryDocumentRepository) Save(_ context.Context, doc ports.CollaborativeDocument) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.docs[doc.ID] = doc // struct copy → snapshot semantics
	return nil
}

// FindByID retrieves a document by ID. Returns nil (not an error) when not found.
func (r *InMemoryDocumentRepository) FindByID(_ context.Context, id string) (*ports.CollaborativeDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	doc, ok := r.docs[id]
	if !ok {
		return nil, nil
	}
	return &doc, nil
}

// List returns all documents in arbitrary order.
func (r *InMemoryDocumentRepository) List(_ context.Context) ([]ports.CollaborativeDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ports.CollaborativeDocument, 0, len(r.docs))
	for _, doc := range r.docs {
		out = append(out, doc)
	}
	return out, nil
}

// Compile-time assertion.
var _ ports.DocumentRepository = (*InMemoryDocumentRepository)(nil)
