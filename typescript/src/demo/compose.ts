import makeGraphAuthorizer from "../adapters/authz/makeGraphAuthorizer.ts";
import makeInMemoryTupleStore from "../adapters/authz/makeInMemoryTupleStore.ts";
import { alice, bob, casey, roadmapDocument, seedRelationshipTuples } from "./fixtures.ts";

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
