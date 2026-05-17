import type { CollaborativeDocument, DocumentId } from "./document.js";

export interface DocumentRepository {
  save(document: CollaborativeDocument): Promise<void>;
  findById(id: DocumentId): Promise<CollaborativeDocument | undefined>;
  list(): Promise<readonly CollaborativeDocument[]>;
}

export class InMemoryDocumentRepository implements DocumentRepository {
  private readonly documents = new Map<DocumentId, CollaborativeDocument>();

  async save(document: CollaborativeDocument): Promise<void> {
    this.documents.set(document.id, cloneDocument(document));
  }

  async findById(id: DocumentId): Promise<CollaborativeDocument | undefined> {
    const document = this.documents.get(id);
    return document ? cloneDocument(document) : undefined;
  }

  async list(): Promise<readonly CollaborativeDocument[]> {
    return [...this.documents.values()].map(cloneDocument);
  }
}

function cloneDocument(document: CollaborativeDocument): CollaborativeDocument {
  return { ...document };
}
