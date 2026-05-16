import { document, subjectSet, team, tuple, user, workspace } from "../authz/types.js";
import type { TupleKey } from "../authz/types.js";

export const alice = user("alice");
export const bob = user("bob");
export const chandra = user("chandra");
export const platform = team("platform");
export const acme = workspace("acme");
export const roadmap = document("roadmap");

export function tutorialTuples(): readonly TupleKey[] {
  return [
    tuple(platform, "member", alice),
    tuple(acme, "editor", subjectSet(platform, "member")),
    tuple(acme, "viewer", bob),
    tuple(roadmap, "workspace", acme)
  ];
}
