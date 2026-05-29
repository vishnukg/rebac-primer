// Documents service composition root.
//
// Wires together:
//   AuthzServiceClient  — HTTP client that calls the AuthZ service
//   DemoTokenVerifier   — authn adapter (swap for JWT verifier in production)
//   InMemoryDocumentRepository
//   DocumentsDomain     — create/read/update operations
//   DocumentsHttpHandler + DocumentsHttpServer
//
// The AuthZ service URL comes from AUTHZ_URL (default: http://127.0.0.1:4100).
// The documents service listens on DOCUMENTS_PORT (default: 4000).

import makeAuthzServiceClient from "./adapters/authz/makeAuthzServiceClient.ts";
import makeDemoTokenVerifier from "./adapters/authn/makeDemoTokenVerifier.ts";
import makeInMemoryDocumentRepository from "./adapters/db/makeInMemoryDocumentRepository.ts";
import makeDocumentsHttpHandler from "./adapters/http/makeDocumentsHttpHandler.ts";
import makeDocumentsHttpServer from "./adapters/http/makeDocumentsHttpServer.ts";
import makeDocuments from "./core/domain/makeDocuments.ts";

type DocumentsServiceCfg = {
    port?:     number;
    authzUrl?: string;
    tokens?:   Record<string, { sub: string; scopes: string[] }>;
};

const composeDocumentsService = ({
    port     = readPort(process.env.DOCUMENTS_PORT, 4000),
    authzUrl = process.env.AUTHZ_URL ?? "http://127.0.0.1:4100",
    tokens   = {},
}: DocumentsServiceCfg = {}) => {
    const authzClient   = makeAuthzServiceClient({ baseUrl: authzUrl });
    const authenticator = makeDemoTokenVerifier({ tokens });
    const repository    = makeInMemoryDocumentRepository();
    const documents     = makeDocuments({ repository, authzClient });
    const handler = makeDocumentsHttpHandler({ authenticator, documents });
    const server  = makeDocumentsHttpServer({ handler });

    const listen = (onReady: (port: number) => Promise<void>) => {
        // Bind all interfaces (0.0.0.0), not just loopback, so the service is
        // reachable across containers and via Docker published ports. Matches the
        // Go server's ":port" bind.
        server.listen(port, "0.0.0.0", () => {
            onReady(port).catch(err => {
                console.error("Startup error:", err);
                process.exit(1);
            });
        });
    };

    return { listen, documents };
};

const readPort = (value: string | undefined, fallback: number): number => {
    if (!value?.trim()) return fallback;
    const p = Number(value);
    if (!Number.isInteger(p) || p < 1 || p > 65_535) throw new Error(`Invalid port: ${value}`);
    return p;
};

export default composeDocumentsService;
