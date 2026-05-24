// Graph-based ReBAC evaluator — the authz service's local implementation.
//
// Answers "does user X have relation R on object O?" by traversing the stored
// relationship tuples.  Reads from TupleRepository (the driven port).
//
// Responsibilities:
//   • Wrap a request in trace + visited state, return CheckResult
//   • Recursive relation check with cycle detection
//   • Tuple lookup: direct match and subject-set resolution
//   • Workspace inheritance: documents inherit owner/editor/viewer from workspace
//
// Permission rules (which relations imply which) live in permissionModel.ts.

import {
    isObjectOfType, isSubjectSet, parseObject, parseSubjectSet,
} from "../../../shared/rebac.ts";
import type {
    CheckRequest, CheckResult, RebacObject, Relation, SubjectSet,
} from "../../../shared/rebac.ts";
import type { TupleRepository } from "../../core/ports/tupleRepository.ts";
import { DOCUMENT_RULES, TEAM_RULES, WORKSPACE_RULES, type ImpliedBy } from "./permissionModel.ts";

type GraphEvaluatorCfg = {
    repository: TupleRepository;
};

// Tracks (object#relation) pairs currently on the call stack — prevents cycles.
type VisitKey = `${RebacObject}#${Relation}`;

const makeGraphEvaluator = ({ repository }: GraphEvaluatorCfg) => {
    const evaluate = (request: CheckRequest): CheckResult => {
        const trace = [
            `Check whether ${request.user} has ${request.relation} on ${request.object}`,
        ];
        const allowed = hasRelation(
            request.user, request.object, request.relation, trace, new Set(),
        );
        trace.push(allowed ? "Result: allowed" : "Result: denied");
        return { allowed, trace };
    };

    // ── Traversal ─────────────────────────────────────────────────────────────

    const hasRelation = (
        user:     RebacObject<"user">,
        object:   RebacObject,
        relation: Relation,
        trace:    string[],
        visited:  Set<VisitKey>,
    ): boolean => {
        const key: VisitKey = `${object}#${relation}`;
        if (visited.has(key)) {
            trace.push(`Already evaluated ${key}; stop this branch`);
            return false;
        }
        visited.add(key);

        if (hasTuple(user, object, relation, trace, visited)) return true;

        const { type } = parseObject(object);
        if (type === "team")      return expandByRules(TEAM_RULES,      user, object, relation, trace, visited);
        if (type === "workspace") return expandByRules(WORKSPACE_RULES, user, object, relation, trace, visited);
        if (type === "document")  return expandDocument(user, object, relation, trace, visited);
        return false;
    };

    // ── Tuple lookup ──────────────────────────────────────────────────────────

    const hasTuple = (
        user:     RebacObject<"user">,
        object:   RebacObject,
        relation: Relation,
        trace:    string[],
        visited:  Set<VisitKey>,
    ): boolean => {
        if (repository.has(object, relation, user)) {
            trace.push(`Found direct tuple (${object}, ${relation}, ${user})`);
            return true;
        }
        for (const t of repository.findByObjectRelation(object, relation)) {
            if (isSubjectSet(t.user) && subjectSetContains(user, t.user, trace, visited)) {
                trace.push(`Found subject-set tuple (${object}, ${relation}, ${t.user})`);
                return true;
            }
        }
        return false;
    };

    const subjectSetContains = (
        user:    RebacObject<"user">,
        subject: SubjectSet,
        trace:   string[],
        visited: Set<VisitKey>,
    ): boolean => {
        const { object, relation } = parseSubjectSet(subject);
        trace.push(`Resolve subject set ${subject}: does it contain ${user}?`);
        return hasRelation(user, object, relation, trace, visited);
    };

    // ── Permission model expansion ─────────────────────────────────────────────

    const expandByRules = (
        rules:    ImpliedBy,
        user:     RebacObject<"user">,
        object:   RebacObject,
        relation: Relation,
        trace:    string[],
        visited:  Set<VisitKey>,
    ): boolean => {
        for (const implied of rules[relation] ?? []) {
            trace.push(`${object} ${relation} includes ${implied}`);
            if (hasRelation(user, object, implied, trace, visited)) return true;
        }
        return false;
    };

    // Documents use DOCUMENT_RULES plus workspace inheritance.
    const expandDocument = (
        user:     RebacObject<"user">,
        object:   RebacObject,
        relation: Relation,
        trace:    string[],
        visited:  Set<VisitKey>,
    ): boolean => {
        if (expandByRules(DOCUMENT_RULES, user, object, relation, trace, visited)) return true;

        if (relation === "owner" || relation === "editor" || relation === "viewer") {
            for (const parent of repository.findByObjectRelation(object, "workspace")) {
                trace.push(`${object} ${relation} can inherit ${relation} from ${parent.user}`);
                if (
                    isObjectOfType(parent.user, "workspace") &&
                    hasRelation(user, parent.user, relation, trace, visited)
                ) return true;
            }
        }
        return false;
    };

    return { evaluate };
};

export default makeGraphEvaluator;
