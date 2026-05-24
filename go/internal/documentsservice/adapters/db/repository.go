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
// It stores snapshots — mutations to the caller's value after Save do not affect
// the stored copy.
type InMemoryDocumentRepository struct {
	mu   sync.RWMutex
	docs map[string]ports.Document
}

// New creates an empty repository.
func New() *InMemoryDocumentRepository {
	return &InMemoryDocumentRepository{
		docs: make(map[string]ports.Document),
	}
}

// Save stores or replaces a document (idempotent on ID).
func (r *InMemoryDocumentRepository) Save(_ context.Context, doc ports.Document) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.docs[doc.ID] = doc // struct copy — snapshot semantics
	return nil
}

// FindByID retrieves a document by ID. Returns nil (not an error) when not found.
func (r *InMemoryDocumentRepository) FindByID(_ context.Context, id string) (*ports.Document, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	doc, ok := r.docs[id]
	if !ok {
		return nil, nil
	}
	return &doc, nil
}

// List returns all documents in arbitrary order.
func (r *InMemoryDocumentRepository) List(_ context.Context) ([]ports.Document, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ports.Document, 0, len(r.docs))
	for _, doc := range r.docs {
		out = append(out, doc)
	}
	return out, nil
}

// Compile-time assertion.
var _ ports.DocumentRepository = (*InMemoryDocumentRepository)(nil)
