import type { RebacObject } from "../../ports/authz.ts";

// ── Domain types ──────────────────────────────────────────────────────────────

export type DocumentId = string;

export type CollaborativeDocument = {
    id:        DocumentId;
    title:     string;
    body:      string;
    workspace: RebacObject<"workspace">;
    updatedBy: RebacObject<"user">;
};

export type CreateDocumentInput = {
    id:        DocumentId;
    title:     string;
    body:      string;
    workspace: RebacObject<"workspace">;
    actor:     RebacObject<"user">;
};

export type ReadDocumentInput = {
    id:    DocumentId;
    actor: RebacObject<"user">;
};

export type UpdateDocumentInput = {
    id:    DocumentId;
    body:  string;
    actor: RebacObject<"user">;
};

// ── Function types ────────────────────────────────────────────────────────────

export type CreateDocumentFn = (input: CreateDocumentInput) => Promise<CollaborativeDocument>;
export type ReadDocumentFn   = (input: ReadDocumentInput) => Promise<CollaborativeDocument>;
export type UpdateDocumentFn = (input: UpdateDocumentInput) => Promise<CollaborativeDocument>;

// ── Ports ─────────────────────────────────────────────────────────────────────

// Driven port — what the domain needs from a data store.
// makeInMemoryDocumentRepository is the adapter.
export interface DocumentRepository {
    save:     (document: CollaborativeDocument) => Promise<void>;
    findById: (id: DocumentId) => Promise<CollaborativeDocument | undefined>;
}

// Driving port — the interface the outside world uses to operate on documents.
// makeHttpHandler is the adapter that translates HTTP calls into calls on this.
export interface Documents {
    create: CreateDocumentFn;
    read:   ReadDocumentFn;
    update: UpdateDocumentFn;
}

// ── Domain errors ─────────────────────────────────────────────────────────────
//
// Tagged errors for safe discrimination at the HTTP boundary.
// The HTTP adapter maps each name to a status code (404, 403).
//
// Pattern: type + factory function, no `class` keyword.
// Type guards (isXxxError) let callers narrow the type after catching.

export type DocumentNotFoundError = Error & { readonly name: "DocumentNotFoundError" };

export const DocumentNotFoundError = (id: DocumentId): DocumentNotFoundError =>
    Object.assign(new Error(`Document not found: ${id}`), {
        name: "DocumentNotFoundError" as const,
    });

export const isDocumentNotFoundError = (e: unknown): e is DocumentNotFoundError =>
    e instanceof Error && e.name === "DocumentNotFoundError";

export type ForbiddenError = Error & { readonly name: "ForbiddenError" };

export const ForbiddenError = (message: string): ForbiddenError =>
    Object.assign(new Error(message), { name: "ForbiddenError" as const });

export const isForbiddenError = (e: unknown): e is ForbiddenError =>
    e instanceof Error && e.name === "ForbiddenError";
