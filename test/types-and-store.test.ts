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
  it("given_domain_ids_when_building_objects_and_subject_sets_then_openfga_strings_are_consistent", () => {
    // Arrange
    const userId = "workspaceEditor";
    const documentId = "roadmapDocument";
    const teamId = "platformTeam";

    // Act
    const workspaceEditor = user(userId);
    const roadmapDocument = document(documentId);
    const platformMembers = subjectSet(team(teamId), "member");

    // Assert
    expect(workspaceEditor).toBe("user:workspaceEditor");
    expect(roadmapDocument).toBe("document:roadmapDocument");
    expect(platformMembers).toBe("team:platformTeam#member");
  });

  it("given_object_id_with_colons_when_parsing_object_then_type_and_full_id_are_returned", () => {
    // Arrange
    const objectId = user("github:123");

    // Act
    const parsed = parseObject(objectId);

    // Assert
    expect(parsed).toEqual({ type: "user", id: "github:123" });
  });

  it("given_team_subject_set_when_parsing_subject_set_then_object_and_relation_are_returned", () => {
    // Arrange
    const members = subjectSet(team("platformTeam"), "member");

    // Act
    const parsed = parseSubjectSet(members);

    // Assert
    expect(parsed).toEqual({ object: "team:platformTeam", relation: "member" });
  });

  it("given_invalid_strings_when_parsing_rebac_ids_then_errors_are_thrown", () => {
    // Arrange
    const invalidObjectId = "not-an-object";
    const invalidSubjectSet = "workspace:productWorkspace#viewer";

    // Act
    const parseObjectAction = () => parseObject(invalidObjectId);
    const parseSubjectSetAction = () => parseSubjectSet(invalidSubjectSet);

    // Assert
    expect(parseObjectAction).toThrow("Invalid OpenFGA object id");
    expect(parseSubjectSetAction).toThrow("Invalid subject set");
  });
});

describe("MemoryTupleStore", () => {
  it("given_tuple_store_when_writing_finding_and_deleting_tuple_then_store_reflects_changes", () => {
    // Arrange
    const workspaceTuple = tuple(document("roadmapDocument"), "workspace", workspace("productWorkspace"));
    const store = new MemoryTupleStore([workspaceTuple]);

    // Act
    const existsBeforeDelete = store.has(document("roadmapDocument"), "workspace", workspace("productWorkspace"));
    const foundBeforeDelete = store.findByObjectRelation(document("roadmapDocument"), "workspace");
    store.delete(workspaceTuple);
    const existsAfterDelete = store.has(document("roadmapDocument"), "workspace", workspace("productWorkspace"));

    // Assert
    expect(existsBeforeDelete).toBe(true);
    expect(foundBeforeDelete).toEqual([workspaceTuple]);
    expect(existsAfterDelete).toBe(false);
  });
});
