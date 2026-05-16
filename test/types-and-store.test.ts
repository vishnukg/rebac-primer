import { describe, expect, it } from "vitest";
import { MemoryTupleStore } from "../src/authz/memory-store.js";
import {
  document,
  parseObject,
  parseSubjectSet,
  subjectSet,
  team,
  tuple,
  user,
  workspace
} from "../src/authz/types.js";

describe("typed OpenFGA helpers", () => {
  it("builds object ids and subject sets consistently", () => {
    expect(user("alice")).toBe("user:alice");
    expect(document("roadmap")).toBe("document:roadmap");
    expect(subjectSet(team("platform"), "member")).toBe("team:platform#member");
  });

  it("parses object ids that contain additional colons", () => {
    expect(parseObject(user("github:123"))).toEqual({ type: "user", id: "github:123" });
  });

  it("parses subject sets into object and relation parts", () => {
    expect(parseSubjectSet(subjectSet(team("platform"), "member"))).toEqual({
      object: "team:platform",
      relation: "member"
    });
  });
});

describe("MemoryTupleStore", () => {
  it("writes, finds, and deletes tuples", () => {
    const workspaceTuple = tuple(document("roadmap"), "workspace", workspace("acme"));
    const store = new MemoryTupleStore([workspaceTuple]);

    expect(store.has(document("roadmap"), "workspace", workspace("acme"))).toBe(true);
    expect(store.findByObjectRelation(document("roadmap"), "workspace")).toEqual([workspaceTuple]);

    store.delete(workspaceTuple);

    expect(store.has(document("roadmap"), "workspace", workspace("acme"))).toBe(false);
  });
});
