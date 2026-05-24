// Graph-based authorizer — local implementation for learning and tests.
//
// Answers "does user X have relation R on object O?" by traversing the stored
// relationship tuples.  This is the same question OpenFGA answers remotely.
//
// Responsibilities of this file:
//   • Entry point: wraps a request in trace + visited state, returns CheckResult
//   • Traversal: recursive relation check with cycle detection
//   • Tuple lookup: direct match and subject-set resolution against the TupleStore
//   • Workspace inheritance: documents inherit owner/editor/viewer from their workspace
//
// The permission rules (which relations imply which) live in permissionModel.ts.

import { isObjectOfType, isSubjectSet, parseObject, parseSubjectSet } from "../../core/index.ts";
import type {
    Authorizer,
    CheckRequest,
    CheckResult,
    RebacObject,
    Relation,
    SubjectSet,
    TupleStore,
} from "../../core/index.ts";
import { DOCUMENT_RULES, TEAM_RULES, WORKSPACE_RULES, type ImpliedBy } from "./permissionModel.ts";

type GraphAuthorizerCfg = {
    tupleStore: Pick<TupleStore, "has" | "findByObjectRelation">;
};

// Tracks which (object, relation) pairs are currently being evaluated to detect cycles.
type VisitKey = `${RebacObject}#${Relation}`;

// ── Factory ───────────────────────────────────────────────────────────────────

const makeGraphAuthorizer = ({ tupleStore }: GraphAuthorizerCfg): Authorizer => {
    // ── Entry point ───────────────────────────────────────────────────────────

    const check = async (request: CheckRequest): Promise<CheckResult> => {
        const trace = [
            `Check whether ${request.user} has ${request.relation} on ${request.object}`,
        ];
        const allowed = hasRelation(request.user, request.object, request.relation, trace, new Set());
        trace.push(allowed ? "Result: allowed" : "Result: denied");
        return { allowed, trace };
    };

    // ── Traversal ─────────────────────────────────────────────────────────────
    //
    // 1. Skip if already evaluated this (object, relation) pair (cycle guard).
    // 2. Check for a stored tuple (direct or via a subject-set group).
    // 3. Expand implied relations from the permission model for this object type.

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

        if (hasTuple(user, object, relation, trace, visited)) return true;

        const { type } = parseObject(object);
        if (type === "team")      return expandByRules(TEAM_RULES, user, object, relation, trace, visited);
        if (type === "workspace") return expandByRules(WORKSPACE_RULES, user, object, relation, trace, visited);
        if (type === "document")  return expandDocument(user, object, relation, trace, visited);
        return false;
    };

    // ── Tuple lookup ──────────────────────────────────────────────────────────
    //
    // Checks the TupleStore for:
    //   • a direct (object, relation, user) tuple
    //   • a subject-set tuple, e.g. team:platform#member, where the user is a member

    const hasTuple = (
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

    // ── Permission model expansion ─────────────────────────────────────────────
    //
    // Expands a relation using an ImpliedBy rules table (from permissionModel.ts).
    // If relation R is implied by [S, T], check S then T on the same object.

    const expandByRules = (
        rules: ImpliedBy,
        user: RebacObject<"user">,
        object: RebacObject,
        relation: Relation,
        trace: string[],
        visited: Set<VisitKey>,
    ): boolean => {
        for (const implied of rules[relation] ?? []) {
            trace.push(`${object} ${relation} includes ${implied}`);
            if (hasRelation(user, object, implied, trace, visited)) return true;
        }
        return false;
    };

    // Documents use DOCUMENT_RULES plus one extra rule: owner/editor/viewer can be
    // inherited from the document's parent workspace via a "workspace" tuple.
    const expandDocument = (
        user: RebacObject<"user">,
        object: RebacObject,
        relation: Relation,
        trace: string[],
        visited: Set<VisitKey>,
    ): boolean => {
        if (expandByRules(DOCUMENT_RULES, user, object, relation, trace, visited)) return true;

        if (relation === "owner" || relation === "editor" || relation === "viewer") {
            for (const parent of tupleStore.findByObjectRelation(object, "workspace")) {
                trace.push(`${object} ${relation} can inherit ${relation} from ${parent.user}`);
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

    return { check };
};

export default makeGraphAuthorizer;
