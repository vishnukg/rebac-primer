import type { DocumentRepository } from "../../core/ports/documentRepository.ts";
import type { CollaborativeDocument, DocumentId } from "../../core/domain/types.ts";

const makeInMemoryDocumentRepository = (): DocumentRepository => {
    const store = new Map<DocumentId, CollaborativeDocument>();
    return {
        save:     async doc  => { store.set(doc.id, { ...doc }); },
        findById: async id   => store.get(id),
    };
};

export default makeInMemoryDocumentRepository;
