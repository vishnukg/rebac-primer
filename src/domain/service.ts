import { document as documentObject } from "../authz/types.js";
import type { Authorizer } from "../authz/types.js";
import {
  type CollaborativeDocument,
  type CreateDocumentInput,
  DocumentNotFoundError,
  ForbiddenError,
  type UpdateDocumentInput
} from "./document.js";
import type { DocumentRepository } from "./repository.js";

export class DocumentService {
  constructor(
    private readonly repository: DocumentRepository,
    private readonly authorizer: Authorizer
  ) {}

  async create(input: CreateDocumentInput): Promise<CollaborativeDocument> {
    const decision = await this.authorizer.check({
      user: input.actor,
      relation: "editor",
      object: input.workspace
    });

    if (!decision.allowed) {
      throw new ForbiddenError(`${input.actor} cannot create documents in ${input.workspace}`);
    }

    const created: CollaborativeDocument = {
      id: input.id,
      title: input.title,
      body: input.body,
      workspace: input.workspace,
      updatedBy: input.actor
    };
    await this.repository.save(created);
    return created;
  }

  async read(id: string, actor: `user:${string}`): Promise<CollaborativeDocument> {
    const existing = await this.requireDocument(id);
    const decision = await this.authorizer.check({
      user: actor,
      relation: "can_read",
      object: documentObject(id)
    });

    if (!decision.allowed) {
      throw new ForbiddenError(`${actor} cannot read document:${id}`);
    }

    return existing;
  }

  async update(input: UpdateDocumentInput): Promise<CollaborativeDocument> {
    const existing = await this.requireDocument(input.id);
    const decision = await this.authorizer.check({
      user: input.actor,
      relation: "can_edit",
      object: documentObject(input.id)
    });

    if (!decision.allowed) {
      throw new ForbiddenError(`${input.actor} cannot edit document:${input.id}`);
    }

    const updated = { ...existing, body: input.body, updatedBy: input.actor };
    await this.repository.save(updated);
    return updated;
  }

  private async requireDocument(id: string): Promise<CollaborativeDocument> {
    const existing = await this.repository.findById(id);
    if (!existing) {
      throw new DocumentNotFoundError(id);
    }

    return existing;
  }
}
