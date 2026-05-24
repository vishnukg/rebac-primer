// Permission model — pure data tables.
//
// Defines which relations are implied by others on the SAME object type.
// The graph evaluator reads these during traversal.
//
// Reading the tables: key = relation being checked, value = stronger relations
// that satisfy it.  WORKSPACE_RULES.viewer = ["editor"] means "workspace viewer
// is satisfied by workspace editor".

import type { Relation } from "../../../shared/rebac.ts";

export type ImpliedBy = Partial<Record<Relation, readonly Relation[]>>;

// team.admin implies team.member
export const TEAM_RULES: ImpliedBy = {
    member: ["admin"],
};

// workspace.owner implies workspace.editor implies workspace.viewer
export const WORKSPACE_RULES: ImpliedBy = {
    editor: ["owner"],
    viewer: ["editor"],
};

// document role hierarchy + computed permissions.
// owner/editor/viewer can ALSO be inherited from the parent workspace —
// that logic lives in makeGraphEvaluator.ts (requires a tuple lookup).
export const DOCUMENT_RULES: ImpliedBy = {
    can_read:    ["viewer"],
    can_comment: ["viewer"],
    can_edit:    ["editor"],
    can_delete:  ["owner"],
    viewer:      ["editor"],
    editor:      ["owner"],
};
