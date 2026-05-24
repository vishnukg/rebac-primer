export { default as makeCreateDocument } from "./makeCreateDocument.ts";
export { default as makeDocuments }      from "./makeDocuments.ts";
export { default as makeReadDocument }   from "./makeReadDocument.ts";
export { default as makeUpdateDocument } from "./makeUpdateDocument.ts";
export {
    DocumentNotFoundError,
    ForbiddenError,
    isDocumentNotFoundError,
    isForbiddenError,
} from "./types.ts";
export type {
    CollaborativeDocument,
    CreateDocumentFn,
    CreateDocumentInput,
    DocumentId,
    DocumentRepository,
    Documents,
    ReadDocumentFn,
    ReadDocumentInput,
    UpdateDocumentFn,
    UpdateDocumentInput,
} from "./types.ts";
