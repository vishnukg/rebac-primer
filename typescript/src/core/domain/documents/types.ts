import type { RebacObject } from "../../ports/authz.ts";

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

export type UpdateDocumentInput = {
    id:    DocumentId;
    body:  string;
    actor: RebacObject<"user">;
};

export type CreateDocumentFn = (input: CreateDocumentInput) => Promise<CollaborativeDocument>;
export type ReadDocumentFn   = (id: DocumentId, actor: RebacObject<"user">) => Promise<CollaborativeDocument>;
export type UpdateDocumentFn = (input: UpdateDocumentInput) => Promise<CollaborativeDocument>;

// Driven port — what the domain requires from a data store.
// makeInMemoryDocumentRepository is the adapter.
export interface DocumentRepository {
    save:     (document: CollaborativeDocument) => Promise<void>;
    findById: (id: DocumentId) => Promise<CollaborativeDocument | undefined>;
}

// Driving port — what the outside world calls into the domain.
// makeHttpHandler is the adapter that translates HTTP into calls on this interface.
export interface Documents {
    create: CreateDocumentFn;
    read:   ReadDocumentFn;
    update: UpdateDocumentFn;
}

export class DocumentNotFoundError extends Error {
    constructor(id: DocumentId) {
        super(`Document not found: ${id}`);
    }
}

export class ForbiddenError extends Error {
    constructor(message: string) {
        super(message);
    }
}
