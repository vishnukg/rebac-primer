// Documents service entrypoint.
//
// Starts the document management service. Depends on the AuthZ service
// being reachable at AUTHZ_URL (default: http://127.0.0.1:4100).
//
// Ports:
//   GET   /whoami
//   POST  /documents
//   GET   /documents/:id
//   PATCH /documents/:id
//
// Port: DOCUMENTS_PORT (default 4000)

import composeDocumentsService from "./compose.ts";
import { demoTokens, seedDocuments } from "../demo/fixtures.ts";

// The demo document is passed as seed config; the composition root creates it on
// startup (once the server is up and authz is reachable) and returns just
// { listen } — it never hands the domain back out. See docs/adr/0001.
const { listen } = composeDocumentsService({
    tokens:        demoTokens,
    seedDocuments: seedDocuments(),
});

listen(port => {
    console.log(`Documents service → http://127.0.0.1:${port}`);
    console.log(`  GET   /whoami`);
    console.log(`  POST  /documents`);
    console.log(`  GET   /documents/:id`);
    console.log(`  PATCH /documents/:id`);
    console.log("\nSeeded: document:roadmapDocument");
    console.log("Demo tokens:");
    console.log("  demo-token-alice  (platform team → editor)");
    console.log("  demo-token-bob    (workspace viewer → can read)");
    console.log("  demo-token-casey  (no relationships → denied)");
});
