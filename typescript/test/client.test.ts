import { describe, expect, it } from "vitest";
import { makeHttpDocumentsClient, makeTerminalClient } from "../src/modules/client/index.ts";
import type { DocumentsClient, Fetcher, QuestionTerminal } from "../src/modules/client/index.ts";

describe("makeHttpDocumentsClient", () => {
  it("reads documents through the HTTP API", async () => {
    const fetcher: Fetcher = async url => {
      expect(url.toString()).toBe("http://server.test/documents/roadmapDocument?actorId=alice");
      return new Response(JSON.stringify({
        document: {
          id:        "roadmapDocument",
          title:     "Roadmap",
          body:      "v1",
          workspace: "workspace:productWorkspace",
          updatedBy: "user:alice",
        },
      }), { status: 200, headers: { "content-type": "application/json" } });
    };
    const client = makeHttpDocumentsClient({ baseUrl: "http://server.test", fetcher });

    const document = await client.readDocument("roadmapDocument", "alice");

    expect(document.id).toBe("roadmapDocument");
  });
});

describe("makeTerminalClient", () => {
  it("runs the read workflow", async () => {
    const writes: string[] = [];
    const answers = ["1", "bob", "3"];
    const terminal: QuestionTerminal = {
      question: async () => {
        const answer = answers.shift();
        if (!answer) throw new Error("No answer arranged");
        return answer;
      },
    };
    const client: DocumentsClient = {
      health: async () => true,
      whoami: async () => "user:bob",
      readDocument: async () => ({
        id:        "roadmapDocument",
        title:     "Roadmap",
        body:      "Read the tutorial",
        workspace: "workspace:productWorkspace",
        updatedBy: "user:alice",
      }),
      updateDocument: async () => {
        throw new Error("Update was not expected");
      },
    };

    await makeTerminalClient({ client, terminal, write: message => writes.push(message) }).run();

    expect(writes).toContain("\nRoadmap");
    expect(writes).toContain("Read the tutorial");
  });
});
