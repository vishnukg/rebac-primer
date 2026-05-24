import { describe, expect, it } from "vitest";
import makeHttpDocumentsClient from "../src/documents-service/adapters/client/makeHttpDocumentsClient.ts";
import makeTerminalClient from "../src/documents-service/adapters/client/makeTerminalClient.ts";
import type { DocumentsClient, Fetcher } from "../src/documents-service/adapters/client/makeHttpDocumentsClient.ts";
import type { QuestionTerminal } from "../src/documents-service/adapters/client/makeTerminalClient.ts";

describe("makeHttpDocumentsClient", () => {
    it("reads a document using a bearer token", async () => {
        const fetcher: Fetcher = async (url, init) => {
            expect(url.toString()).toBe("http://server.test/documents/roadmapDocument");
            expect((init?.headers as Record<string, string>)?.authorization).toBe(
                "Bearer demo-token-alice",
            );
            return new Response(
                JSON.stringify({
                    document: {
                        id:        "roadmapDocument",
                        title:     "Roadmap",
                        body:      "v1",
                        workspace: "workspace:productWorkspace",
                        updatedBy: "user:alice",
                    },
                }),
                { status: 200, headers: { "content-type": "application/json" } },
            );
        };
        const client = makeHttpDocumentsClient({ baseUrl: "http://server.test", fetcher });

        const doc = await client.readDocument("roadmapDocument", "demo-token-alice");

        expect(doc.id).toBe("roadmapDocument");
    });
});

describe("makeTerminalClient", () => {
    it("authenticates then reads a document", async () => {
        const writes: string[] = [];
        // Answers: token, then choose "2" (read), document id, then "4" (exit)
        const answers = ["demo-token-bob", "2", "roadmapDocument", "4"];
        const terminal: QuestionTerminal = {
            question: async () => {
                const answer = answers.shift();
                if (!answer) throw new Error("No answer arranged");
                return answer;
            },
        };
        const client: DocumentsClient = {
            health:         async () => true,
            whoami:         async () => "user:bob",
            createDocument: async () => { throw new Error("Not expected"); },
            readDocument:   async () => ({
                id:        "roadmapDocument",
                title:     "Roadmap",
                body:      "Read the tutorial",
                workspace: "workspace:productWorkspace",
                updatedBy: "user:alice",
            }),
            updateDocument: async () => { throw new Error("Not expected"); },
        };

        await makeTerminalClient({
            client,
            terminal,
            write: message => writes.push(message),
        }).run();

        expect(writes).toContain("Authenticated as: user:bob");
        expect(writes).toContain("\nRoadmap");
        expect(writes).toContain("Read the tutorial");
    });
});
