// ── Composition root ──────────────────────────────────────────────────────────
//
// This is where the application is wired together for the HTTP server.
// Each make*() call runs once at startup; the result is used for every request.
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
import { makeDocuments } from "../core/index.ts";
import type { Authenticator, Authorizer, DocumentRepository } from "../core/index.ts";

type ServerAppCfg = {
    port:          number;
    authenticator: Authenticator; // driven port — verifies bearer tokens
    authorizer:    Authorizer;    // driven port — checks ReBAC permissions
    repository:    DocumentRepository; // driven port — stores documents
};

const makeServerApp = ({ port, authenticator, authorizer, repository }: ServerAppCfg) => {
    const documents = makeDocuments({ repository, authorizer });
    const handler = makeHttpHandler({ authenticator, documents });
    const server = makeHttpServer({ handler });

    return { port, server, documents };
};

export default makeServerApp;
