// Demo scenario fixtures — used by the server entry point and by tests.
// Defines the alice/bob/casey scenario described in the README.

import { document, subjectSet, team, tuple, user, workspace } from "../core/index.ts";
import type { CreateDocumentInput, TokenClaims, TupleKey } from "../core/index.ts";

// ── Actors ────────────────────────────────────────────────────────────────────

export const alice = user("alice");
export const bob   = user("bob");
export const casey = user("casey");

// ── Objects ───────────────────────────────────────────────────────────────────

export const platformTeam     = team("platformTeam");
export const productWorkspace = workspace("productWorkspace");
export const roadmapDocument  = document("roadmapDocument");

// ── Demo tokens ───────────────────────────────────────────────────────────────

export const demoTokens: Record<string, TokenClaims> = {
    "demo-token-alice": { sub: "alice", scopes: ["documents:read", "documents:write"] },
    "demo-token-bob":   { sub: "bob",   scopes: ["documents:read"] },
    "demo-token-casey": { sub: "casey", scopes: ["documents:read"] },
};

// ── Relationship tuples ───────────────────────────────────────────────────────

// alice is a member of platformTeam
// platformTeam#member are editors of productWorkspace
// bob is a viewer of productWorkspace
// roadmapDocument lives in productWorkspace
export const seedRelationshipTuples = (): TupleKey[] => [
    tuple(platformTeam, "member", alice),
    tuple(productWorkspace, "editor", subjectSet(platformTeam, "member")),
    tuple(productWorkspace, "viewer", bob),
    tuple(roadmapDocument, "workspace", productWorkspace),
];

// ── Seed document ─────────────────────────────────────────────────────────────

export const seedRoadmapDocument: CreateDocumentInput = {
    id:        "roadmapDocument",
    title:     "Roadmap",
    body:      "Initial roadmap document",
    workspace: productWorkspace,
    actor:     alice,
};
