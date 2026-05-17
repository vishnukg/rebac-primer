import { document, subjectSet, team, tuple, user, workspace } from "../authz/types.js";
import type { TupleKey } from "../authz/types.js";

export const alice = user("alice");
export const bob = user("bob");
export const casey = user("casey");
export const platformTeam = team("platformTeam");
export const productWorkspace = workspace("productWorkspace");
export const roadmapDocument = document("roadmapDocument");

export function seedRelationshipTuples(): readonly TupleKey[] {
  return [
    tuple(platformTeam, "member", alice),
    tuple(productWorkspace, "editor", subjectSet(platformTeam, "member")),
    tuple(productWorkspace, "viewer", bob),
    tuple(roadmapDocument, "workspace", productWorkspace)
  ];
}
