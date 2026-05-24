export { default as makeCreateDocument }  from "./makeCreateDocument.ts";
export { default as makeDocuments }       from "./makeDocuments.ts";
export { default as makeReadDocument }    from "./makeReadDocument.ts";
export { default as makeRequireAllowed }  from "./makeRequireAllowed.ts";
export { default as makeUpdateDocument }  from "./makeUpdateDocument.ts";
export { DocumentNotFoundError, ForbiddenError } from "./types.ts";
export type {
  CollaborativeDocument,
  CreateDocumentFn,
  CreateDocumentInput,
  DocumentId,
  DocumentOperationDeps,
  DocumentRepository,
  Documents,
  ReadDocumentFn,
  RequireAllowedFn,
  UpdateDocumentFn,
  UpdateDocumentInput,
} from "./types.ts";
