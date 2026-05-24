// Tests for the graph evaluator — the ReBAC traversal engine inside the authz service.
import { describe, expect, it } from "vitest";
import makeInMemoryTupleRepository from "../src/authz-service/adapters/db/makeInMemoryTupleRepository.ts";
import makeGraphEvaluator from "../src/authz-service/adapters/graph/makeGraphEvaluator.ts";
import { subjectSet, team, tuple, user, workspace, document } from "../src/shared/rebac.ts";
import { seedPolicyTuples, platformTeam, productWorkspace, alice, bob, casey } from "../src/demo/fixtures.ts";

const makeEvaluator = (extra = []) => {
    const repository = makeInMemoryTupleRepository([...seedPolicyTuples(), ...extra]);
    return makeGraphEvaluator({ repository });
};

const roadmapDoc = document("roadmapDocument");

// Simulate the tuple the documents service writes on create.
const docWorkspaceTuple = tuple(roadmapDoc, "workspace", productWorkspace);

describe("makeGraphEvaluator", () => {
    it("allows alice to edit via team → workspace → document", () => {
        const ev = makeEvaluator([docWorkspaceTuple]);
        const { allowed, trace } = ev.evaluate({
            user: alice, relation: "can_edit", object: roadmapDoc,
        });
        expect(allowed).toBe(true);
        expect(trace.some(l => l.includes("team:platformTeam#member"))).toBe(true);
    });

    it("allows bob to read via workspace viewer inheritance", () => {
        const ev = makeEvaluator([docWorkspaceTuple]);
        const { allowed } = ev.evaluate({
            user: bob, relation: "can_read", object: roadmapDoc,
        });
        expect(allowed).toBe(true);
    });

    it("denies bob from editing (viewer, not editor)", () => {
        const ev = makeEvaluator([docWorkspaceTuple]);
        const { allowed } = ev.evaluate({
            user: bob, relation: "can_edit", object: roadmapDoc,
        });
        expect(allowed).toBe(false);
    });

    it("denies casey who has no path in the graph", () => {
        const ev = makeEvaluator([docWorkspaceTuple]);
        const { allowed } = ev.evaluate({
            user: casey, relation: "can_read", object: roadmapDoc,
        });
        expect(allowed).toBe(false);
    });

    it("handles a cycle without hanging", () => {
        const cyclicDoc = document("cyclicDoc");
        const ev = makeEvaluator([
            tuple(cyclicDoc, "workspace", cyclicDoc as unknown as `workspace:${string}`),
            tuple(cyclicDoc, "viewer", bob),
        ]);
        const { allowed } = ev.evaluate({
            user: bob, relation: "can_read", object: cyclicDoc,
        });
        expect(allowed).toBe(true); // direct viewer tuple still matches
    });

    it("resolves workspace owner → document owner via workspace inheritance", () => {
        const ownerDoc = document("ownerDoc");
        const ev = makeEvaluator([
            tuple(ownerDoc, "workspace", productWorkspace),
            tuple(productWorkspace, "owner", alice),
        ]);
        const { allowed } = ev.evaluate({
            user: alice, relation: "can_delete", object: ownerDoc,
        });
        expect(allowed).toBe(true);
    });
});
