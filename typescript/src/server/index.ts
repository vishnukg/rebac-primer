// Server entrypoint — starts both the AuthZ service and the Documents service.
//
// The AuthZ service is the authorization engine.  It stores relationship tuples
// and evaluates the ReBAC graph.
//
// The Documents service is the example product that uses the AuthZ service.
// It calls POST /check to verify permissions and POST /tuples when documents
// are created.
//
// Ports (defaults):
//   AuthZ service:     4100  (override with AUTHZ_PORT)
//   Documents service: 4000  (override with DOCUMENTS_PORT)

import makeAuthzService from "../authz-service/compose.ts";
import makeDocumentsService from "../documents-service/compose.ts";
import { demoTokens, seedPolicyTuples } from "../demo/fixtures.ts";

const authz     = makeAuthzService({ seedTuples: seedPolicyTuples() });
const documents = makeDocumentsService({ tokens: demoTokens });

authz.server.listen(authz.port, "127.0.0.1", () => {
    console.log(`AuthZ service     → http://127.0.0.1:${authz.port}`);
    console.log(`  POST /check     — check a permission`);
    console.log(`  POST /tuples    — write relationship tuples`);
    console.log(`  DELETE /tuples  — remove relationship tuples`);
    console.log(`  GET  /tuples    — list tuples (audit)`);
});

documents.server.listen(documents.port, "127.0.0.1", async () => {
    console.log(`\nDocuments service → http://127.0.0.1:${documents.port}`);
    console.log(`  GET  /whoami`);
    console.log(`  POST /documents`);
    console.log(`  GET  /documents/:id`);
    console.log(`  PATCH /documents/:id`);

    // Seed a demo document through the documents service.
    // This call goes: documents.create → authz.check (can alice create?)
    //                                  → repo.save
    //                                  → authz.writeTuples (workspace + owner)
    try {
        await documents.documents.create({
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
