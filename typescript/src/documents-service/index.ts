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
import { demoTokens } from "../demo/fixtures.ts";

const { listen, documents } = composeDocumentsService({ tokens: demoTokens });

listen(async port => {
    console.log(`Documents service → http://127.0.0.1:${port}`);
    console.log(`  GET   /whoami`);
    console.log(`  POST  /documents`);
    console.log(`  GET   /documents/:id`);
    console.log(`  PATCH /documents/:id`);

    // Seed a demo document so the server is ready to explore immediately.
    // This write goes: domain.create → authz.check → repo.save → authz.writeTuples
    try {
        await documents.create({
            id:        "roadmapDocument",
            title:     "Roadmap",
            body:      "Initial roadmap document",
            workspace: "workspace:productWorkspace",
            actor:     "user:alice",
        });
        console.log("\nSeeded: document:roadmapDocument");
        console.log("Demo tokens:");
        console.log("  demo-token-alice  (platform team → editor)");
        console.log("  demo-token-bob    (workspace viewer → can read)");
        console.log("  demo-token-casey  (no relationships → denied)");
    } catch (e) {
        console.error("Failed to seed demo document:", e);
    }
});
