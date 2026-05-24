import { describe, expect, it } from "vitest";
import makeGraphAuthorizer from "../src/adapters/authz/makeGraphAuthorizer.ts";
import makeInMemoryTupleStore from "../src/adapters/authz/makeInMemoryTupleStore.ts";
import {
    document,
    parseObject,
    parseSubjectSet,
    subjectSet,
    team,
    tuple,
    user,
} from "../src/core/index.ts";
import {
    alice,
    bob,
    casey,
    platformTeam,
    roadmapDocument,
    seedRelationshipTuples,
} from "../src/demo/fixtures.ts";

describe("ReBAC object helpers", () => {
    it("builds and parses OpenFGA-style ids", () => {
        expect(user("alice")).toBe("user:alice");
        expect(document("roadmap")).toBe("document:roadmap");
        expect(subjectSet(team("platform"), "member")).toBe("team:platform#member");
        expect(parseObject(user("github:123"))).toEqual({ type: "user", id: "github:123" });
        expect(parseSubjectSet(subjectSet(team("platform"), "member"))).toEqual({
            object:   "team:platform",
            relation: "member",
        });
    });
});

describe("makeGraphAuthorizer", () => {
    it("allows alice to edit through team membership and workspace inheritance", async () => {
        const tupleStore = makeInMemoryTupleStore({ seed: seedRelationshipTuples() });
        const authorizer = makeGraphAuthorizer({ tupleStore });

        const result = await authorizer.check({
            user:     alice,
            relation: "can_edit",
            object:   roadmapDocument,
        });

        expect(result.allowed).toBe(true);
        expect(result.trace).toContain(
            "Resolve subject set team:platformTeam#member: does it contain user:alice?",
        );
        expect(result.trace).toContain(
            "document:roadmapDocument editor can inherit editor from workspace:productWorkspace",
        );
    });

    it("lets bob read as a workspace viewer but denies editing", async () => {
        const authorizer = makeGraphAuthorizer({
            tupleStore: makeInMemoryTupleStore({ seed: seedRelationshipTuples() }),
        });

        await expect(
            authorizer.check({ user: bob, relation: "can_read", object: roadmapDocument }),
        ).resolves.toMatchObject({ allowed: true });

        await expect(
            authorizer.check({ user: bob, relation: "can_edit", object: roadmapDocument }),
        ).resolves.toMatchObject({ allowed: false });
    });

    it("treats team admins as team members", async () => {
        const authorizer = makeGraphAuthorizer({
            tupleStore: makeInMemoryTupleStore({
                seed: [...seedRelationshipTuples(), tuple(platformTeam, "admin", casey)],
            }),
        });

        const result = await authorizer.check({
            user:     casey,
            relation: "member",
            object:   platformTeam,
        });

        expect(result.allowed).toBe(true);
        expect(result.trace).toContain("team:platformTeam member includes admin");
    });

    it("stops cycles without denying unrelated direct grants", async () => {
        const cyclicDocument = document("cyclic");
        const authorizer = makeGraphAuthorizer({
            tupleStore: makeInMemoryTupleStore({
                seed: [
                    tuple(cyclicDocument, "workspace", cyclicDocument),
                    tuple(cyclicDocument, "viewer", bob),
                ],
            }),
        });

        const result = await authorizer.check({
            user:     bob,
            relation: "can_read",
            object:   cyclicDocument,
        });

        expect(result.allowed).toBe(true);
    });
});
