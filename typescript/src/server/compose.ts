// ── Composition root ──────────────────────────────────────────────────────────
//
// This is where the application is wired together for the HTTP server.
// Concrete adapters are constructed once at startup; the result is used for
// every request.
//
// The flow for a protected request:
//
//   HTTP request
//     → makeHttpHandler        (adapter) — routes the request
//       → authenticator        (port)    — "who is calling?" (authn)
//       → documents.read/update/create   (domain) — executes business logic
//           → authorizer       (port)    — "is this caller allowed?" (authz)
//           → repository       (port)    — reads/writes documents
//
// authn answers: who are you?  (verifies the token → gives back a user identity)
// authz answers: what can you do? (checks the relationship graph → allowed/denied)

import makeHttpHandler from "../adapters/http/makeHttpHandler.ts";
import makeHttpServer from "../adapters/http/makeHttpServer.ts";
import makeGraphAuthorizer from "../adapters/authz/makeGraphAuthorizer.ts";
import makeInMemoryTupleStore from "../adapters/authz/makeInMemoryTupleStore.ts";
import makeDemoTokenVerifier from "../adapters/authn/makeDemoTokenVerifier.ts";
import makeInMemoryDocumentRepository from "../adapters/db/makeInMemoryDocumentRepository.ts";
import { makeDocuments } from "../core/index.ts";
import {
    demoTokens,
    seedRelationshipTuples,
    seedRoadmapDocument,
} from "../demo/fixtures.ts";

type ServerAppCfg = {
    port?: number;
};

const makeServerApp = ({ port = readPort(process.env.PORT, 4000) }: ServerAppCfg = {}) => {
    const tupleStore   = makeInMemoryTupleStore({ seed: seedRelationshipTuples() });
    const authorizer   = makeGraphAuthorizer({ tupleStore });
    const authenticator = makeDemoTokenVerifier({ tokens: demoTokens });
    const repository = makeInMemoryDocumentRepository();
    const documents = makeDocuments({ repository, authorizer });
    const handler = makeHttpHandler({ authenticator, documents });
    const server = makeHttpServer({ handler });

    return { port, server, documents, seedDocument: seedRoadmapDocument };
};

const readPort = (value: string | undefined, fallback: number): number => {
    if (value === undefined || value.trim() === "") return fallback;
    const portValue = Number(value);
    if (!Number.isInteger(portValue) || portValue < 1 || portValue > 65_535) {
        throw new Error(`Invalid PORT: ${value}`);
    }
    return portValue;
};

export default makeServerApp;
