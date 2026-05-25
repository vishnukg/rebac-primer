// AuthZ service composition root.
//
// Wires together:
//   TupleRepository  (in-memory, seeded with workspace/team policies)
//   GraphEvaluator   (ReBAC traversal, reads from TupleRepository)
//   AuthzDomain      (check + writeTuples + deleteTuples + listTuples)
//   AuthzHttpHandler (routes POST /check, POST /tuples, etc.)
//   AuthzHttpServer  (Node HTTP server)
//
// The seed tuples represent the workspace/team policies that a platform team
// would configure by calling POST /tuples on this service.
// Document-level tuples (workspace relation, owner) are written by the
// documents service at document-creation time.

import makeInMemoryTupleRepository from "./adapters/db/makeInMemoryTupleRepository.ts";
import makeGraphEvaluator from "./adapters/graph/makeGraphEvaluator.ts";
import makeAuthzHttpHandler from "./adapters/http/makeAuthzHttpHandler.ts";
import makeAuthzHttpServer from "./adapters/http/makeAuthzHttpServer.ts";
import makeAuthzDomain from "./core/domain/makeAuthzDomain.ts";

type AuthzServiceCfg = {
    port?:        number;
    seedTuples?:  import("../shared/rebac.ts").TupleKey[];
};

const composeAuthzService = ({
    port        = readPort(process.env.AUTHZ_PORT, 4100),
    seedTuples  = [],
}: AuthzServiceCfg = {}) => {
    const repository = makeInMemoryTupleRepository({ seed: seedTuples });
    const evaluator  = makeGraphEvaluator({ repository });
    const domain     = makeAuthzDomain({ repository, evaluator });
    const handler = makeAuthzHttpHandler({ authz: domain });
    const server  = makeAuthzHttpServer({ handler });

    const listen = (onReady: (port: number) => void) => {
        server.listen(port, "127.0.0.1", () => onReady(port));
    };

    return { listen, domain };
};

const readPort = (value: string | undefined, fallback: number): number => {
    if (!value?.trim()) return fallback;
    const p = Number(value);
    if (!Number.isInteger(p) || p < 1 || p > 65_535) throw new Error(`Invalid port: ${value}`);
    return p;
};

export default composeAuthzService;
