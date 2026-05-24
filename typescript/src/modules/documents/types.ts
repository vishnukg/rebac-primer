import type { Authorizer, RebacObject, Relation } from "../authz/index.ts";

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
export type ReadDocumentFn = (id: DocumentId, actor: RebacObject<"user">) => Promise<CollaborativeDocument>;
export type UpdateDocumentFn = (input: UpdateDocumentInput) => Promise<CollaborativeDocument>;

export interface DocumentRepository {
  save:     (document: CollaborativeDocument) => Promise<void>;
  findById: (id: DocumentId) => Promise<CollaborativeDocument | undefined>;
  list:     () => Promise<CollaborativeDocument[]>;
}

export interface Documents {
  create: CreateDocumentFn;
  read:   ReadDocumentFn;
  update: UpdateDocumentFn;
}

export type DocumentOperationDeps = {
  repository: DocumentRepository;
  authorizer: Authorizer;
};

export type RequireAllowedFn = (
  actor: RebacObject<"user">,
  relation: Relation,
  object: RebacObject,
  action: string,
) => Promise<void>;

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
