import type { CollaborativeDocument, DocumentId, DocumentRepository } from "../documents/index.ts";

const makeInMemoryDocumentRepository = (): DocumentRepository => {
  const documents = new Map<DocumentId, CollaborativeDocument>();

  const save = async (document: CollaborativeDocument): Promise<void> => {
    documents.set(document.id, cloneDocument(document));
  };

  const findById = async (id: DocumentId): Promise<CollaborativeDocument | undefined> => {
    const found = documents.get(id);
    return found ? cloneDocument(found) : undefined;
  };

  const list = async (): Promise<CollaborativeDocument[]> =>
    [...documents.values()].map(cloneDocument);

  return { save, findById, list };
};

const cloneDocument = (document: CollaborativeDocument): CollaborativeDocument => ({ ...document });

export default makeInMemoryDocumentRepository;
