import type { CollaborativeDocument, DocumentId } from "../domain/types.ts";

// Driven port — persistence for documents.
export interface DocumentRepository {
    save:     (document: CollaborativeDocument) => Promise<void>;
    findById: (id: DocumentId) => Promise<CollaborativeDocument | undefined>;
}
