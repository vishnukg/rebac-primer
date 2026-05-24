import type { RebacObject } from "../../../shared/rebac.ts";

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

// ── Function types ─────────────────────────────────────────────────────────────

export type CreateDocumentFn = (input: CreateDocumentInput) => Promise<CollaborativeDocument>;
export type ReadDocumentFn   = (input: ReadDocumentInput)   => Promise<CollaborativeDocument>;
export type UpdateDocumentFn = (input: UpdateDocumentInput) => Promise<CollaborativeDocument>;

// ── Driving port ───────────────────────────────────────────────────────────────

export interface Documents {
    create: CreateDocumentFn;
    read:   ReadDocumentFn;
    update: UpdateDocumentFn;
}

// ── Domain errors ──────────────────────────────────────────────────────────────

export type DocumentNotFoundError = Error & { readonly name: "DocumentNotFoundError" };
export const DocumentNotFoundError = (id: DocumentId): DocumentNotFoundError =>
    Object.assign(new Error(`Document not found: ${id}`), { name: "DocumentNotFoundError" as const });
export const isDocumentNotFoundError = (e: unknown): e is DocumentNotFoundError =>
    e instanceof Error && e.name === "DocumentNotFoundError";

export type ForbiddenError = Error & { readonly name: "ForbiddenError" };
export const ForbiddenError = (message: string): ForbiddenError =>
    Object.assign(new Error(message), { name: "ForbiddenError" as const });
export const isForbiddenError = (e: unknown): e is ForbiddenError =>
    e instanceof Error && e.name === "ForbiddenError";
