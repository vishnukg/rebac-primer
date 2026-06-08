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
import readPort from "../shared/readPort.ts";
import type { CreateDocumentInput } from "./core/index.ts";

type DocumentsServiceCfg = {
    port?:          number;
    authzUrl?:      string;
    tokens?:        Record<string, { sub: string; scopes: string[] }>;
    seedDocuments?: CreateDocumentInput[];
};

const composeDocumentsService = ({
    port          = readPort(process.env.DOCUMENTS_PORT, 4000),
    authzUrl      = process.env.AUTHZ_URL ?? "http://127.0.0.1:4100",
    tokens        = {},
    seedDocuments = [],
}: DocumentsServiceCfg = {}) => {
    const authzClient   = makeAuthzServiceClient({ baseUrl: authzUrl });
    const authenticator = makeDemoTokenVerifier({ tokens });
    const repository    = makeInMemoryDocumentRepository();
    const documents     = makeDocuments({ repository, authzClient });
    const handler = makeDocumentsHttpHandler({ authenticator, documents });
    const server  = makeDocumentsHttpServer({ handler });

    // Seeding a document is a full domain op (create → authz.check → repo.save →
    // authz.writeTuples), so it can only run once the server is up and the authz
    // service is reachable — not at construction time like authz's tuple seed.
    // Doing it here lets this root return just { listen } instead of exposing the
    // domain so the entry point can seed it (see docs/adr/0001).
    const seed = async () => {
        for (const doc of seedDocuments) {
            try {
                await documents.create(doc);
            } catch (err) {
                console.error(`Failed to seed document ${doc.id}:`, err);
            }
        }
    };

    const listen = (onReady: (port: number) => void) => {
        // Bind all interfaces (0.0.0.0), not just loopback, so the service is
        // reachable across containers and via Docker published ports. Matches the
        // Go server's ":port" bind.
        server.listen(port, "0.0.0.0", () => {
            void seed().then(() => onReady(port));
        });
    };

    return { listen };
};

export default composeDocumentsService;
