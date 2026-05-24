import { OpenFgaClient } from "@openfga/sdk";
import type { Authorizer, TupleKey, WriteTuplesFn } from "./types.ts";

type OpenFgaAuthorizerCfg = {
  apiUrl:                string;
  storeId:               string;
  authorizationModelId?: string;
};

export type OpenFgaAuthorizer = Authorizer & {
  writeTuples: WriteTuplesFn;
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
      writes: tuples.map(tupleKey => ({
        user:     tupleKey.user,
        relation: tupleKey.relation,
        object:   tupleKey.object,
      })),
    });
  };

  return { check, writeTuples };
};

export default makeOpenFgaAuthorizer;
