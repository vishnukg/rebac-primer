import { GraphAuthorizer } from "../authz/graph-authorizer.js";
import { MemoryTupleStore, type TupleStore } from "../authz/memory-store.js";
import type { Authorizer } from "../authz/types.js";
import { tuple } from "../authz/types.js";
import { InMemoryDocumentRepository } from "../domain/repository.js";
import { DocumentService, type DocumentWorkflow } from "../domain/service.js";
import {
  productWorkspace,
  roadmapDocument,
  seedRelationshipTuples,
  workspaceEditor
} from "../testing/fixtures.js";

export type AppServices = Readonly<{
  documents: DocumentWorkflow;
  authorizer: Authorizer;
  tupleStore: TupleStore;
}>;

export async function createServices(): Promise<AppServices> {
  const tupleStore = new MemoryTupleStore(seedRelationshipTuples());
  const authorizer = new GraphAuthorizer(tupleStore);
  const repository = new InMemoryDocumentRepository();
  const documents = new DocumentService(repository, authorizer);

  await documents.create({
    id: "roadmapDocument",
    title: "Roadmap",
    body: "Initial roadmap document",
    workspace: productWorkspace,
    actor: workspaceEditor
  });
  tupleStore.write(tuple(roadmapDocument, "workspace", productWorkspace));

  return { documents, authorizer, tupleStore };
}
