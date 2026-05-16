import { GraphAuthorizer } from "./authz/graph-authorizer.js";
import { MemoryTupleStore } from "./authz/memory-store.js";
import { roadmap, alice, bob, chandra, tutorialTuples } from "./testing/fixtures.js";

const authorizer = new GraphAuthorizer(new MemoryTupleStore(tutorialTuples()));

for (const actor of [alice, bob, chandra]) {
  const result = await authorizer.check({
    user: actor,
    relation: "can_edit",
    object: roadmap
  });

  console.log(`${actor} can_edit ${roadmap}: ${result.allowed}`);
  console.log(result.trace.map((line) => `  ${line}`).join("\n"));
}
