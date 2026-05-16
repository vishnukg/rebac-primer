import type { CollaborativeDocument, DocumentId } from "./document.js";

export interface DocumentRepository {
  save(document: CollaborativeDocument): Promise<void>;
  findById(id: DocumentId): Promise<CollaborativeDocument | undefined>;
  list(): Promise<readonly CollaborativeDocument[]>;
}

export class InMemoryDocumentRepository implements DocumentRepository {
  private readonly documents = new Map<DocumentId, CollaborativeDocument>();

  async save(document: CollaborativeDocument): Promise<void> {
    this.documents.set(document.id, document);
  }

  async findById(id: DocumentId): Promise<CollaborativeDocument | undefined> {
    return this.documents.get(id);
  }

  async list(): Promise<readonly CollaborativeDocument[]> {
    return [...this.documents.values()];
  }
}
