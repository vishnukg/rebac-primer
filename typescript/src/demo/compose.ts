import { makeGraphAuthorizer, makeInMemoryTupleStore } from "../modules/authz/index.ts";
import { alice, bob, casey, roadmapDocument, seedRelationshipTuples } from "../modules/fixtures/index.ts";

const makeDemoApp = () => {
  const tupleStore = makeInMemoryTupleStore({ seed: seedRelationshipTuples() });
  const authorizer = makeGraphAuthorizer({ tupleStore });

  return {
    authorizer,
    actors:   [alice, bob, casey],
    document: roadmapDocument,
  };
};

export default makeDemoApp;
