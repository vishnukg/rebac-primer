import type { CollaborativeDocument, DocumentId, DocumentRepository } from "../../core/index.ts";

// Stores snapshots of documents (shallow copies), so callers cannot mutate stored state.
const makeInMemoryDocumentRepository = (): DocumentRepository => {
    const store = new Map<DocumentId, CollaborativeDocument>();

    const save = async (document: CollaborativeDocument): Promise<void> => {
        store.set(document.id, { ...document });
    };

    const findById = async (id: DocumentId): Promise<CollaborativeDocument | undefined> => {
        const found = store.get(id);
        return found ? { ...found } : undefined;
    };

    return { save, findById };
};

export default makeInMemoryDocumentRepository;
