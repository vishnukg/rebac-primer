// Kept for backwards compatibility — see test/graphEvaluator.test.ts for the
// full graph traversal suite.  Tests here verify the shared rebac.ts helpers.
import { describe, expect, it } from "vitest";
import { parseObject, parseSubjectSet, user, team, workspace, document, tuple, subjectSet } from
    "../src/shared/rebac.ts";

describe("ReBAC object helpers", () => {
    it("builds and parses OpenFGA-style ids", () => {
        expect(user("alice")).toBe("user:alice");
        expect(team("platform")).toBe("team:platform");
        expect(workspace("prod")).toBe("workspace:prod");
        expect(document("roadmap")).toBe("document:roadmap");

        expect(parseObject("user:alice")).toEqual({ type: "user", id: "alice" });
        expect(parseObject("workspace:prod:v2")).toEqual({ type: "workspace", id: "prod:v2" });

        expect(() => parseObject("bad")).toThrow();
        expect(() => parseObject("unknown:x")).toThrow();
    });

    it("builds and parses subject sets", () => {
        const ss = subjectSet(team("platform"), "member");
        expect(ss).toBe("team:platform#member");
        expect(parseSubjectSet(ss)).toEqual({ object: "team:platform", relation: "member" });
    });

    it("builds tuples", () => {
        expect(tuple(document("d"), "viewer", user("alice"))).toEqual({
            object: "document:d", relation: "viewer", user: "user:alice",
        });
    });
});
