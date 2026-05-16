import { document, subjectSet, team, tuple, user, workspace } from "../authz/types.js";
import type { TupleKey } from "../authz/types.js";

export const workspaceEditor = user("workspaceEditor");
export const workspaceViewer = user("workspaceViewer");
export const outsideCollaborator = user("outsideCollaborator");
export const platformTeam = team("platformTeam");
export const productWorkspace = workspace("productWorkspace");
export const roadmapDocument = document("roadmapDocument");

export function seedRelationshipTuples(): readonly TupleKey[] {
  return [
    tuple(platformTeam, "member", workspaceEditor),
    tuple(productWorkspace, "editor", subjectSet(platformTeam, "member")),
    tuple(productWorkspace, "viewer", workspaceViewer),
    tuple(roadmapDocument, "workspace", productWorkspace)
  ];
}
