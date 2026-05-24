import {
    isObjectOfType,
    isSubjectSet,
    parseObject,
    parseSubjectSet,
} from "../../core/index.ts";
import type {
    Authorizer,
    CheckRequest,
    CheckResult,
    RebacObject,
    Relation,
    SubjectSet,
    TupleStore,
} from "../../core/index.ts";

type GraphAuthorizerCfg = {
    tupleStore: Pick<TupleStore, "has" | "findByObjectRelation">;
};

// Internal key used to detect cycles during graph traversal
type VisitKey = `${RebacObject}#${Relation}`;

// ── Permission inheritance rules ──────────────────────────────────────────────
//
// These mirror the OpenFGA model in adapters/authz/model.ts.
// When a user asks for relation X, we also check whether they have any of
// the listed implied relations — if so, the original check passes too.
//
// Example: if a user has can_edit, they can_read (via viewer → editor chain).
//
const DOCUMENT_IMPLIED_PERMISSIONS: Partial<Record<Relation, Relation[]>> = {
    can_read:    ["viewer"],          // can_read  = viewer
    can_comment: ["viewer"],          // can_comment = viewer
    can_edit:    ["editor"],          // can_edit  = editor
    can_delete:  ["owner"],           // can_delete = owner
    viewer:      ["editor"],          // viewer ⊆ editor
    editor:      ["owner"],           // editor ⊆ owner
};

// ── Authorizer ────────────────────────────────────────────────────────────────

const makeGraphAuthorizer = ({ tupleStore }: GraphAuthorizerCfg): Authorizer => {
    const check = async (request: CheckRequest): Promise<CheckResult> => {
        const trace = [
            `Check whether ${request.user} has ${request.relation} on ${request.object}`,
        ];
        const allowed = hasRelation(
            request.user,
            request.object,
            request.relation,
            trace,
            new Set(),
        );
        trace.push(allowed ? "Result: allowed" : "Result: denied");
        return { allowed, trace };
    };

    // Main recursive check. At each step:
    //   1. Look for a direct tuple in the store
    //   2. Expand implied relations based on object type
    const hasRelation = (
        user: RebacObject<"user">,
        object: RebacObject,
        relation: Relation,
        trace: string[],
        visited: Set<VisitKey>,
    ): boolean => {
        const visitKey: VisitKey = `${object}#${relation}`;
        if (visited.has(visitKey)) {
            trace.push(`Already evaluated ${visitKey}; stop this branch`);
            return false;
        }
        visited.add(visitKey);

        if (hasDirectOrSubjectSet(user, object, relation, trace, visited)) return true;

        const { type } = parseObject(object);
        if (type === "team")      return expandTeam(user, object, relation, trace, visited);
        if (type === "workspace") return expandWorkspace(user, object, relation, trace, visited);
        if (type === "document")  return expandDocument(user, object, relation, trace, visited);
        return false;
    };

    // team.member includes team.admin — admins are always members
    const expandTeam = (
        user: RebacObject<"user">,
        object: RebacObject,
        relation: Relation,
        trace: string[],
        visited: Set<VisitKey>,
    ): boolean => {
        if (relation !== "member") return false;
        trace.push("team.member includes team.admin");
        return hasRelation(user, object, "admin", trace, visited);
    };

    // workspace.viewer ⊆ workspace.editor ⊆ workspace.owner
    const expandWorkspace = (
        user: RebacObject<"user">,
        object: RebacObject,
        relation: Relation,
        trace: string[],
        visited: Set<VisitKey>,
    ): boolean => {
        if (relation === "editor") {
            trace.push("workspace.editor includes workspace.owner");
            return hasRelation(user, object, "owner", trace, visited);
        }
        if (relation === "viewer") {
            trace.push("workspace.viewer includes workspace.editor");
            return hasRelation(user, object, "editor", trace, visited);
        }
        return false;
    };

    // Documents follow DOCUMENT_IMPLIED_PERMISSIONS above, plus they can inherit
    // owner/editor/viewer from their parent workspace via a "workspace" tuple.
    const expandDocument = (
        user: RebacObject<"user">,
        object: RebacObject,
        relation: Relation,
        trace: string[],
        visited: Set<VisitKey>,
    ): boolean => {
        for (const implied of DOCUMENT_IMPLIED_PERMISSIONS[relation] ?? []) {
            trace.push(`document.${relation} includes document.${implied}`);
            if (hasRelation(user, object, implied, trace, visited)) return true;
        }

        // Inherit from workspace: e.g. being a workspace editor also makes you a document editor
        if (relation === "owner" || relation === "editor" || relation === "viewer") {
            for (const parent of tupleStore.findByObjectRelation(object, "workspace")) {
                trace.push(
                    `document.${relation} can inherit workspace.${relation} from ${parent.user}`,
                );
                if (
                    isObjectOfType(parent.user, "workspace") &&
                    hasRelation(user, parent.user, relation, trace, visited)
                ) {
                    return true;
                }
            }
        }

        return false;
    };

    // Check for a direct `(object, relation, user)` tuple, or a subject-set tuple
    // like `team:platform#member` where the user is a member of that team.
    const hasDirectOrSubjectSet = (
        user: RebacObject<"user">,
        object: RebacObject,
        relation: Relation,
        trace: string[],
        visited: Set<VisitKey>,
    ): boolean => {
        if (tupleStore.has(object, relation, user)) {
            trace.push(`Found direct tuple (${object}, ${relation}, ${user})`);
            return true;
        }

        for (const t of tupleStore.findByObjectRelation(object, relation)) {
            if (isSubjectSet(t.user) && subjectSetContains(user, t.user, trace, visited)) {
                trace.push(`Found subject-set tuple (${object}, ${relation}, ${t.user})`);
                return true;
            }
        }

        return false;
    };

    const subjectSetContains = (
        user: RebacObject<"user">,
        subject: SubjectSet,
        trace: string[],
        visited: Set<VisitKey>,
    ): boolean => {
        const { object, relation } = parseSubjectSet(subject);
        trace.push(`Resolve subject set ${subject}: does it contain ${user}?`);
        return hasRelation(user, object, relation, trace, visited);
    };

    return { check };
};

export default makeGraphAuthorizer;
