import { GraphAuthorizer } from "../authz/graph-authorizer.js";
import { MemoryTupleStore } from "../authz/memory-store.js";
import type { Authorizer, RebacObject } from "../authz/types.js";
import { alice, bob, chandra, roadmap, tutorialTuples } from "../testing/fixtures.js";

export type DemoApp = Readonly<{
  authorizer: Authorizer;
  actors: readonly RebacObject<"user">[];
  document: RebacObject<"document">;
}>;

export function createDemoApp(): DemoApp {
  return {
    authorizer: new GraphAuthorizer(new MemoryTupleStore(tutorialTuples())),
    actors: [alice, bob, chandra],
    document: roadmap
  };
}
