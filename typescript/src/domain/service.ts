import { document as documentObject } from "../authz/types.js";
import type { Authorizer, RebacObject, Relation } from "../authz/types.js";
import {
  type CollaborativeDocument,
  type CreateDocumentInput,
  DocumentNotFoundError,
  ForbiddenError,
  type UpdateDocumentInput
} from "./document.js";
import type { DocumentRepository } from "./repository.js";

export interface DocumentOperations {
  create(input: CreateDocumentInput): Promise<CollaborativeDocument>;
  read(id: string, actor: RebacObject<"user">): Promise<CollaborativeDocument>;
  update(input: UpdateDocumentInput): Promise<CollaborativeDocument>;
}

export class DocumentService implements DocumentOperations {
  constructor(
    private readonly repository: DocumentRepository,
    private readonly authorizer: Authorizer
  ) {}

  async create(input: CreateDocumentInput): Promise<CollaborativeDocument> {
    await this.requireAllowed(input.actor, "editor", input.workspace, "create documents in");

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

  async read(id: string, actor: RebacObject<"user">): Promise<CollaborativeDocument> {
    const existing = await this.requireDocument(id);
    await this.requireAllowed(actor, "can_read", documentObject(id), "read");

    return existing;
  }

  async update(input: UpdateDocumentInput): Promise<CollaborativeDocument> {
    const existing = await this.requireDocument(input.id);
    await this.requireAllowed(input.actor, "can_edit", documentObject(input.id), "edit");

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

  private async requireAllowed(
    actor: RebacObject<"user">,
    relation: Relation,
    object: RebacObject,
    action: string
  ): Promise<void> {
    const decision = await this.authorizer.check({ user: actor, relation, object });

    if (!decision.allowed) {
      throw new ForbiddenError(`${actor} cannot ${action} ${object}`);
    }
  }
}
