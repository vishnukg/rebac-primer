import type { DocumentRepository } from "../../core/ports/documentRepository.ts";
import type { CollaborativeDocument, DocumentId } from "../../core/domain/types.ts";

const makeInMemoryDocumentRepository = (): DocumentRepository => {
    const store = new Map<DocumentId, CollaborativeDocument>();
    return {
        // Both save and findById deep-copy the document so callers cannot mutate
        // the stored value through the reference they hold (snapshot semantics —
        // matches the Go FindByID, which returns a struct copy). structuredClone
        // is a deep clone, so it stays correct if the document grows nested fields.
        save:     async doc  => { store.set(doc.id, structuredClone(doc)); },
        findById: async id   => { const doc = store.get(id); return doc && structuredClone(doc); },
    };
};

export default makeInMemoryDocumentRepository;
