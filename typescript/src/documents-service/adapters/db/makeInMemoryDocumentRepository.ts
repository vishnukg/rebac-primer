import type { DocumentRepository } from "../../core/ports/documentRepository.ts";
import type { CollaborativeDocument, DocumentId } from "../../core/domain/types.ts";

const makeInMemoryDocumentRepository = (): DocumentRepository => {
    const store = new Map<DocumentId, CollaborativeDocument>();
    return {
        // Both save and findById copy the document so callers cannot mutate the
        // stored value through the reference they hold (snapshot semantics —
        // matches the Go FindByID, which returns a struct copy).
        save:     async doc  => { store.set(doc.id, { ...doc }); },
        findById: async id   => { const doc = store.get(id); return doc && { ...doc }; },
    };
};

export default makeInMemoryDocumentRepository;
