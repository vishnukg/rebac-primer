import { GraphAuthorizer } from "../authz/graph-authorizer.js";
import { InMemoryTupleStore } from "../authz/memory-store.js";
import type { Authorizer, RebacObject } from "../authz/types.js";
import {
  casey,
  roadmapDocument,
  seedRelationshipTuples,
  alice,
  bob
} from "../testing/fixtures.js";

export type DemoApp = Readonly<{
  authorizer: Authorizer;
  actors: readonly RebacObject<"user">[];
  document: RebacObject<"document">;
}>;

export function createDemoApp(): DemoApp {
  return {
    authorizer: new GraphAuthorizer(new InMemoryTupleStore(seedRelationshipTuples())),
    actors: [alice, bob, casey],
    document: roadmapDocument
  };
}
