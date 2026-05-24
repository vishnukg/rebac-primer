import { OpenFgaClient } from "@openfga/sdk";
import type { Authorizer, TupleKey } from "../../core/index.ts";

type OpenFgaAuthorizerCfg = {
    apiUrl:                string;
    storeId:               string;
    authorizationModelId?: string;
};

// Extends the base Authorizer with the ability to write new tuples to OpenFGA.
export type OpenFgaAuthorizer = Authorizer & {
    writeTuples: (tuples: TupleKey[]) => Promise<void>;
};

const makeOpenFgaAuthorizer = (cfg: OpenFgaAuthorizerCfg): OpenFgaAuthorizer => {
    const client = new OpenFgaClient(cfg);

    const check: Authorizer["check"] = async request => {
        const response = await client.check({
            user:     request.user,
            relation: request.relation,
            object:   request.object,
        });
        return {
            allowed: response.allowed === true,
            trace:   ["OpenFGA evaluated the relationship graph remotely"],
        };
    };

    const writeTuples = async (tuples: TupleKey[]): Promise<void> => {
        await client.write({
            writes: tuples.map(t => ({ user: t.user, relation: t.relation, object: t.object })),
        });
    };

    return { check, writeTuples };
};

export default makeOpenFgaAuthorizer;
