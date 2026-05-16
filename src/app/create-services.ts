import { GraphAuthorizer } from "../authz/graph-authorizer.js";
import { MemoryTupleStore, type TupleStore } from "../authz/memory-store.js";
import type { Authorizer } from "../authz/types.js";
import { tuple } from "../authz/types.js";
import { InMemoryDocumentRepository } from "../domain/repository.js";
import { DocumentService, type DocumentWorkflow } from "../domain/service.js";
import { acme, alice, roadmap, tutorialTuples } from "../testing/fixtures.js";

export type AppServices = Readonly<{
  documents: DocumentWorkflow;
  authorizer: Authorizer;
  tupleStore: TupleStore;
}>;

export async function createServices(): Promise<AppServices> {
  const tupleStore = new MemoryTupleStore(tutorialTuples());
  const authorizer = new GraphAuthorizer(tupleStore);
  const repository = new InMemoryDocumentRepository();
  const documents = new DocumentService(repository, authorizer);

  await documents.create({
    id: "roadmap",
    title: "Roadmap",
    body: "Initial roadmap",
    workspace: acme,
    actor: alice
  });
  tupleStore.write(tuple(roadmap, "workspace", acme));

  return { documents, authorizer, tupleStore };
}
