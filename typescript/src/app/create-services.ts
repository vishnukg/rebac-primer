import { GraphAuthorizer } from "../authz/graph-authorizer.js";
import { InMemoryTupleStore, type TupleStore } from "../authz/memory-store.js";
import type { Authorizer } from "../authz/types.js";
import { InMemoryDocumentRepository } from "../domain/repository.js";
import { DocumentService, type DocumentOperations } from "../domain/service.js";
import { productWorkspace, seedRelationshipTuples, alice } from "../testing/fixtures.js";

export type AppServices = Readonly<{
  documents: DocumentOperations;
  authorizer: Authorizer;
  tupleStore: TupleStore;
}>;

export async function createServices(): Promise<AppServices> {
  const tupleStore = new InMemoryTupleStore(seedRelationshipTuples());
  const authorizer = new GraphAuthorizer(tupleStore);
  const repository = new InMemoryDocumentRepository();
  const documents = new DocumentService(repository, authorizer);

  await documents.create({
    id: "roadmapDocument",
    title: "Roadmap",
    body: "Initial roadmap document",
    workspace: productWorkspace,
    actor: alice
  });

  return { documents, authorizer, tupleStore };
}
