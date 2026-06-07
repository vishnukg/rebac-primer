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
import makeOpenFgaAuthzService from "./adapters/openfga/makeOpenFgaAuthzService.ts";
import makeAuthzHttpHandler from "./adapters/http/makeAuthzHttpHandler.ts";
import makeAuthzHttpServer from "./adapters/http/makeAuthzHttpServer.ts";
import makeAuthzDomain from "./core/domain/makeAuthzDomain.ts";
import readPort from "../shared/readPort.ts";
import type { AuthzService } from "./core/index.ts";
import type { TupleKey } from "../shared/rebac.ts";

type AuthzServiceCfg = {
    port?:        number;
    seedTuples?:  TupleKey[];
};

// composeAuthzBackend selects the authorization backend from the environment and
// wires it into an AuthzService:
//   AUTHZ_BACKEND=openfga → a real OpenFGA server (OPENFGA_API_URL/STORE_ID/MODEL_ID)
//   otherwise (default)   → the in-process graph evaluator over an in-memory store
// It is a compose* (not a make*) because it builds its own collaborators via make*
// factories and selects a concrete adapter. Both branches return an AuthzService,
// so the HTTP handler below is identical for either.
const composeAuthzBackend = (seedTuples: TupleKey[]): AuthzService => {
    if (process.env.AUTHZ_BACKEND === "openfga") {
        const apiUrl  = process.env.OPENFGA_API_URL ?? "http://127.0.0.1:8080";
        const storeId = process.env.OPENFGA_STORE_ID;
        const modelId = process.env.OPENFGA_MODEL_ID;
        if (!storeId || !modelId) {
            throw new Error(
                "AUTHZ_BACKEND=openfga requires OPENFGA_STORE_ID and OPENFGA_MODEL_ID " +
                "(run deployments/openfga/seed.sh)",
            );
        }
        console.log(`AuthZ backend: openfga (${apiUrl})`);
        return makeOpenFgaAuthzService({ apiUrl, storeId, modelId });
    }
    const repository = makeInMemoryTupleRepository({ seed: seedTuples });
    const evaluator  = makeGraphEvaluator({ repository });
    console.log("AuthZ backend: in-process graph evaluator");
    return makeAuthzDomain({ repository, evaluator });
};

const composeAuthzService = ({
    port        = readPort(process.env.AUTHZ_PORT, 4100),
    seedTuples  = [],
}: AuthzServiceCfg = {}) => {
    const domain  = composeAuthzBackend(seedTuples);
    const handler = makeAuthzHttpHandler({ authz: domain });
    const server  = makeAuthzHttpServer({ handler });

    const listen = (onReady: (port: number) => void) => {
        // Bind all interfaces (0.0.0.0), not just loopback, so the service is
        // reachable across containers and via Docker published ports. Matches the
        // Go server's ":port" bind.
        server.listen(port, "0.0.0.0", () => onReady(port));
    };

    return { listen, domain };
};

export default composeAuthzService;
