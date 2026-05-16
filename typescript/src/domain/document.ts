import type { RebacObject } from "../authz/types.js";

export type DocumentId = string;

export type CollaborativeDocument = Readonly<{
  id: DocumentId;
  title: string;
  body: string;
  workspace: RebacObject<"workspace">;
  updatedBy: RebacObject<"user">;
}>;

export type CreateDocumentInput = Readonly<{
  id: DocumentId;
  title: string;
  body: string;
  workspace: RebacObject<"workspace">;
  actor: RebacObject<"user">;
}>;

export type UpdateDocumentInput = Readonly<{
  id: DocumentId;
  body: string;
  actor: RebacObject<"user">;
}>;

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
