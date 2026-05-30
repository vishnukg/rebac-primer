// Unit tests for the HTTP AuthzClient adapter (makeAuthzServiceClient).
//
// This adapter is the network seam between the documents service and a
// standalone authz service. It has no Go counterpart (the Go documents service
// calls authz in-process), so there is no mirror suite — but the adapter is
// real production code and deserves coverage.
//
// The collaborator is the Fetcher. Here it is a MOCK: each test's fetcher both
// returns a canned Response (stub behaviour) and asserts on the request it
// received (mock behaviour) — URL, method, and JSON body.
import { describe, expect, it } from "vitest";
import makeAuthzServiceClient from "../src/documents-service/adapters/authz/makeAuthzServiceClient.ts";
import { document, tuple } from "../src/shared/rebac.ts";
import { alice } from "./fixtures.ts";

const jsonResponse = (body: unknown, status = 200): Response =>
    new Response(JSON.stringify(body), { status, headers: { "content-type": "application/json" } });

const roadmapDoc = document("roadmapDocument");

describe("makeAuthzServiceClient — check", () => {
    it("POSTs the check request and returns the parsed result", async () => {
        // Arrange: a MOCK fetcher that asserts on the request and returns a result.
        const client = makeAuthzServiceClient({
            baseUrl: "http://authz.test",
            fetcher: async (url, init) => {
                expect(url.toString()).toBe("http://authz.test/check");
                expect(init?.method).toBe("POST");
                expect(JSON.parse(String(init?.body))).toEqual({
                    user: alice,
                    relation: "can_edit",
                    object: roadmapDoc,
                });
                return jsonResponse({ allowed: true, trace: ["Result: allowed"] });
            },
        });

        // Act
        const result = await client.check({ user: alice, relation: "can_edit", object: roadmapDoc });

        // Assert
        expect(result.allowed).toBe(true);
    });

    it("throws when the authz service responds with an error status", async () => {
        // Arrange: a MOCK fetcher returning a 400 with an error body.
        const client = makeAuthzServiceClient({
            baseUrl: "http://authz.test",
            fetcher: async () => jsonResponse({ error: "bad request" }, 400),
        });

        // Act + Assert
        await expect(
            client.check({ user: alice, relation: "can_edit", object: roadmapDoc }),
        ).rejects.toThrow("AuthZ service error: bad request");
    });
});

describe("makeAuthzServiceClient — writeTuples", () => {
    it("POSTs the tuples to /tuples", async () => {
        // Arrange: a MOCK fetcher that captures and asserts on the request body.
        const tuples = [tuple(roadmapDoc, "owner", alice)];
        const client = makeAuthzServiceClient({
            baseUrl: "http://authz.test",
            fetcher: async (url, init) => {
                expect(url.toString()).toBe("http://authz.test/tuples");
                expect(init?.method).toBe("POST");
                expect(JSON.parse(String(init?.body))).toEqual({ tuples });
                return jsonResponse({ written: 1 });
            },
        });

        // Act + Assert (no throw)
        await client.writeTuples(tuples);
    });
});
