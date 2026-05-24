// ── Permission model ──────────────────────────────────────────────────────────
//
// Defines which relations are implied by others on the SAME object.
// "X is implied by Y" means: if you hold Y, you also hold X.
//
// These rules mirror the OpenFGA schema in model.ts but as plain data so
// the graph traversal in makeGraphAuthorizer.ts can read them.
//
// Reading the tables: key = relation being checked, value = stronger relations
// that satisfy it.
//
//   WORKSPACE_RULES.viewer = ["editor"]
//   → "workspace viewer is satisfied by workspace editor"

import type { Relation } from "../../core/index.ts";

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

// document.can_* are satisfied by the named role.
// The role hierarchy (owner → editor → viewer) is also listed here.
// Note: owner/editor/viewer can also be inherited from the parent workspace —
// that logic lives in makeGraphAuthorizer.ts because it requires a tuple lookup.
export const DOCUMENT_RULES: ImpliedBy = {
    can_read:    ["viewer"],
    can_comment: ["viewer"],
    can_edit:    ["editor"],
    can_delete:  ["owner"],
    viewer:      ["editor"],
    editor:      ["owner"],
};
