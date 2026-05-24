import {
    makeGraphPermissionEvaluator,
    makeTupleStoreRelationshipReader,
} from "../adapters/authz/graphEvaluation.ts";
import makeInMemoryTupleStore from "../adapters/authz/makeInMemoryTupleStore.ts";
import { staticAuthorizationPolicy } from "../adapters/authz/graphPolicy.ts";
import type { Authorizer } from "../core/index.ts";
import { alice, bob, casey, roadmapDocument, seedRelationshipTuples } from "./fixtures.ts";

const makeDemoApp = () => {
    const tupleStore = makeInMemoryTupleStore({ seed: seedRelationshipTuples() });
    const relationships = makeTupleStoreRelationshipReader(tupleStore);
    const evaluator = makeGraphPermissionEvaluator({
        relationships,
        policy: staticAuthorizationPolicy,
    });
    const authorizer: Authorizer = {
        check: async request => evaluator.check(request),
    };

    return {
        authorizer,
        actors:   [alice, bob, casey],
        document: roadmapDocument,
    };
};

export default makeDemoApp;
