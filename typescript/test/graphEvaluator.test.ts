// Tests for the graph evaluator — the ReBAC traversal engine inside the authz service.
import { describe, expect, it } from "vitest";
import makeInMemoryTupleRepository from "../src/authz-service/adapters/db/makeInMemoryTupleRepository.ts";
import makeGraphEvaluator from "../src/authz-service/adapters/graph/makeGraphEvaluator.ts";
import { document, tuple } from "../src/shared/rebac.ts";
import { seedPolicyTuples, productWorkspace, alice, bob, casey } from "./fixtures.ts";

// Helper: builds an evaluator pre-seeded with policy tuples plus any extras.
const makeEvaluator = (extra: ReturnType<typeof seedPolicyTuples> = []) => {
    const repository = makeInMemoryTupleRepository([...seedPolicyTuples(), ...extra]);
    return makeGraphEvaluator({ repository });
};

const roadmapDoc = document("roadmapDocument");

// The documents service writes this tuple when a document is created.
// It links the document to its parent workspace, enabling inheritance.
const docWorkspaceTuple = tuple(roadmapDoc, "workspace", productWorkspace);

describe("makeGraphEvaluator", () => {
    it("allows alice to edit via team → workspace → document chain", async () => {
        // Arrange: alice is a platformTeam member; platformTeam is workspace editor;
        //          document inherits from workspace.
        const ev = makeEvaluator([docWorkspaceTuple]);

        // Act
        const { allowed, trace } = await ev.evaluate({
            user: alice, relation: "can_edit", object: roadmapDoc,
        });

        // Assert
        expect(allowed).toBe(true);
        expect(trace.some(l => l.includes("team:platformTeam#member"))).toBe(true);
    });

    it("allows bob to read via workspace viewer inheritance", async () => {
        // Arrange: bob is a direct viewer of productWorkspace.
        const ev = makeEvaluator([docWorkspaceTuple]);

        // Act
        const { allowed } = await ev.evaluate({
            user: bob, relation: "can_read", object: roadmapDoc,
        });

        // Assert
        expect(allowed).toBe(true);
    });

    it("denies bob from editing (viewer role does not satisfy can_edit)", async () => {
        // Arrange
        const ev = makeEvaluator([docWorkspaceTuple]);

        // Act
        const { allowed } = await ev.evaluate({
            user: bob, relation: "can_edit", object: roadmapDoc,
        });

        // Assert
        expect(allowed).toBe(false);
    });

    it("denies casey who has no path in the relationship graph", async () => {
        // Arrange: casey has no tuples at all.
        const ev = makeEvaluator([docWorkspaceTuple]);

        // Act
        const { allowed } = await ev.evaluate({
            user: casey, relation: "can_read", object: roadmapDoc,
        });

        // Assert
        expect(allowed).toBe(false);
    });

    it("handles a cycle in the graph without hanging", async () => {
        // Arrange: a document points back to itself as its workspace (pathological cycle).
        const cyclicDoc = document("cyclicDoc");
        const ev = makeEvaluator([
            tuple(cyclicDoc, "workspace", cyclicDoc as unknown as `workspace:${string}`),
            tuple(cyclicDoc, "viewer", bob),
        ]);

        // Act
        const { allowed } = await ev.evaluate({
            user: bob, relation: "can_read", object: cyclicDoc,
        });

        // Assert: the direct viewer tuple is found before the cycle is traversed.
        expect(allowed).toBe(true);
    });

    it("resolves workspace owner → document can_delete via workspace inheritance", async () => {
        // Arrange: alice is owner of the workspace; document inherits from workspace.
        const ownerDoc = document("ownerDoc");
        const ev = makeEvaluator([
            tuple(ownerDoc, "workspace", productWorkspace),
            tuple(productWorkspace, "owner", alice),
        ]);

        // Act
        const { allowed } = await ev.evaluate({
            user: alice, relation: "can_delete", object: ownerDoc,
        });

        // Assert
        expect(allowed).toBe(true);
    });
});
