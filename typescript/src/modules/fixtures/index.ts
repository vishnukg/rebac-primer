import { document, subjectSet, team, tuple, user, workspace } from "../authz/index.ts";
import type { TokenClaims } from "../authn/index.ts";
import type { TupleKey } from "../authz/index.ts";

export const alice = user("alice");
export const bob = user("bob");
export const casey = user("casey");

export const platformTeam = team("platformTeam");
export const productWorkspace = workspace("productWorkspace");
export const roadmapDocument = document("roadmapDocument");

export const demoTokens: Record<string, TokenClaims> = {
  "demo-token-alice": { sub: "alice", scopes: ["documents:read", "documents:write"] },
  "demo-token-bob":   { sub: "bob", scopes: ["documents:read"] },
  "demo-token-casey": { sub: "casey", scopes: ["documents:read"] },
};

export const seedRelationshipTuples = (): TupleKey[] => [
  tuple(platformTeam, "member", alice),
  tuple(productWorkspace, "editor", subjectSet(platformTeam, "member")),
  tuple(productWorkspace, "viewer", bob),
  tuple(roadmapDocument, "workspace", productWorkspace),
];

export const seedRoadmapDocument = {
  id:        "roadmapDocument",
  title:     "Roadmap",
  body:      "Initial roadmap document",
  workspace: productWorkspace,
  actor:     alice,
};
