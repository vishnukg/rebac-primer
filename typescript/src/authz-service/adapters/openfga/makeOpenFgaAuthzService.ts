// OpenFGA adapter for the AuthZ service.
//
// This implements the same `AuthzService` driving port as composeAuthzDomain (the
// from-scratch graph version), but answers checks and stores tuples in a real
// OpenFGA server instead of the in-process graph evaluator + in-memory store.
// composeAuthzService picks between the two based on AUTHZ_BACKEND, so nothing
// downstream (the HTTP handler, the documents service) changes.
//
// The model and the workspace/team policy tuples are seeded into the OpenFGA
// store out of band (deployments/openfga/seed.sh). Document-level tuples are
// still written at runtime via writeTuples — they just land in OpenFGA.

import { OpenFgaClient } from "@openfga/sdk";
import type { AuthzService } from "../../core/index.ts";
import type {
    CheckRequest, CheckResult, RebacObject, Relation, Subject, TupleKey,
} from "../../../shared/rebac.ts";
import type { TupleFilter } from "../../core/ports/index.ts";

type OpenFgaAuthzServiceCfg = {
    apiUrl:  string;
    storeId: string;
    modelId: string;
};

const makeOpenFgaAuthzService = ({ apiUrl, storeId, modelId }: OpenFgaAuthzServiceCfg): AuthzService => {
    const client = new OpenFgaClient({ apiUrl, storeId, authorizationModelId: modelId });

    const check = async (request: CheckRequest): Promise<CheckResult> => {
        const { allowed } = await client.check({
            user:     request.user,
            relation: request.relation,
            object:   request.object,
        });
        const ok = allowed === true;
        return {
            allowed: ok,
            // OpenFGA returns only allow/deny, so the trace is one synthetic line
            // rather than the step-by-step trace the in-process evaluator builds.
            trace: [`OpenFGA: ${request.user} ${request.relation} ${request.object} -> ${ok}`],
        };
    };

    const writeTuples = async (tuples: TupleKey[]): Promise<void> => {
        if (tuples.length === 0) return;
        await client.writeTuples(
            tuples.map(t => ({ user: t.user, relation: t.relation, object: t.object })),
        );
    };

    const deleteTuples = async (tuples: TupleKey[]): Promise<void> => {
        if (tuples.length === 0) return;
        await client.deleteTuples(
            tuples.map(t => ({ user: t.user, relation: t.relation, object: t.object })),
        );
    };

    const listTuples = async (filter?: TupleFilter): Promise<TupleKey[]> => {
        // Build the read filter with only the fields that are set — exactOptional-
        // PropertyTypes forbids passing `undefined` for the SDK's optional fields.
        const tupleKey: { object?: string; relation?: string } = {};
        if (filter?.object) tupleKey.object = filter.object;
        if (filter?.relation) tupleKey.relation = filter.relation;
        const response = await client.read(tupleKey);
        return (response.tuples ?? []).flatMap(tuple => {
            const key = tuple.key;
            if (!key) return [];
            // Boundary conversion: OpenFGA returns plain strings; brand them back
            // into the repo's typed tuple shape.
            return [{
                object:   key.object as RebacObject,
                relation: key.relation as Relation,
                user:     key.user as Subject,
            }];
        });
    };

    return { check, writeTuples, deleteTuples, listTuples };
};

export default makeOpenFgaAuthzService;
