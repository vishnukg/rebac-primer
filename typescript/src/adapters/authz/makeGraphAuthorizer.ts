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

type AuthzPolicy = {
    inheritedRelations: Partial<Record<Relation, readonly Relation[]>>;
    parent?: {
        tupleRelation: Relation;
        allowedRelations: readonly Relation[];
    };
};

type VisitKey = `${RebacObject}#${Relation}`;

// These rules mirror adapters/authz/model.ts in code that is easy to debug.
// Read each entry as: checking the key relation also checks the listed stronger
// relations on the same object.
const POLICIES: Record<"team" | "workspace" | "document", AuthzPolicy> = {
    team: {
        inheritedRelations: {
            member: ["admin"],
        },
    },
    workspace: {
        inheritedRelations: {
            viewer: ["editor"],
            editor: ["owner"],
        },
    },
    document: {
        inheritedRelations: {
            can_read:    ["viewer"],
            can_comment: ["viewer"],
            can_edit:    ["editor"],
            can_delete:  ["owner"],
            viewer:      ["editor"],
            editor:      ["owner"],
        },
        parent: {
            tupleRelation:    "workspace",
            allowedRelations: ["owner", "editor", "viewer"],
        },
    },
};

const makeGraphAuthorizer = ({ tupleStore }: GraphAuthorizerCfg): Authorizer => {
    const check = async (request: CheckRequest): Promise<CheckResult> => {
        const trace = [
            `Check whether ${request.user} has ${request.relation} on ${request.object}`,
        ];
        const allowed = evaluate(
            request.user,
            request.object,
            request.relation,
            { trace, evaluating: new Set(), memo: new Map() },
        );
        trace.push(allowed ? "Result: allowed" : "Result: denied");
        return { allowed, trace };
    };

    function evaluate(
        user: RebacObject<"user">,
        object: RebacObject,
        relation: Relation,
        state: EvaluationState,
    ): boolean {
        const visitKey: VisitKey = `${object}#${relation}`;
        const cached = state.memo.get(visitKey);
        if (cached !== undefined) return cached;

        if (state.evaluating.has(visitKey)) {
            state.trace.push(`Already evaluating ${visitKey}; stop this cycle`);
            return false;
        }
        state.evaluating.add(visitKey);

        const allowed = evaluateUncached(user, object, relation, state);
        state.evaluating.delete(visitKey);
        state.memo.set(visitKey, allowed);
        return allowed;
    }

    function evaluateUncached(
        user: RebacObject<"user">,
        object: RebacObject,
        relation: Relation,
        state: EvaluationState,
    ): boolean {
        if (hasStoredTuple(user, object, relation, state)) return true;

        const objectType = parseObject(object).type;
        if (objectType === "user") return false;

        const policy = POLICIES[objectType];
        return (
            hasInheritedRelation(user, object, relation, policy, state) ||
            hasParentRelation(user, object, relation, policy, state)
        );
    }

    function hasStoredTuple(
        user: RebacObject<"user">,
        object: RebacObject,
        relation: Relation,
        state: EvaluationState,
    ): boolean {
        if (tupleStore.has(object, relation, user)) {
            state.trace.push(`Found direct tuple (${object}, ${relation}, ${user})`);
            return true;
        }

        for (const tupleKey of tupleStore.findByObjectRelation(object, relation)) {
            if (isSubjectSet(tupleKey.user) && subjectSetContains(user, tupleKey.user, state)) {
                state.trace.push(
                    `Found subject-set tuple (${object}, ${relation}, ${tupleKey.user})`,
                );
                return true;
            }
        }

        return false;
    }

    function hasInheritedRelation(
        user: RebacObject<"user">,
        object: RebacObject,
        relation: Relation,
        policy: AuthzPolicy,
        state: EvaluationState,
    ): boolean {
        for (const strongerRelation of policy.inheritedRelations[relation] ?? []) {
            state.trace.push(`${object} ${relation} includes ${strongerRelation}`);
            if (evaluate(user, object, strongerRelation, state)) return true;
        }
        return false;
    }

    function hasParentRelation(
        user: RebacObject<"user">,
        object: RebacObject,
        relation: Relation,
        policy: AuthzPolicy,
        state: EvaluationState,
    ): boolean {
        const parent = policy.parent;
        if (!parent || !parent.allowedRelations.includes(relation)) return false;

        for (const tupleKey of tupleStore.findByObjectRelation(object, parent.tupleRelation)) {
            state.trace.push(
                `${object} ${relation} can inherit ${relation} from ${tupleKey.user}`,
            );
            if (
                isObjectOfType(tupleKey.user, "workspace") &&
                evaluate(user, tupleKey.user, relation, state)
            ) {
                return true;
            }
        }

        return false;
    }

    function subjectSetContains(
        user: RebacObject<"user">,
        subject: SubjectSet,
        state: EvaluationState,
    ): boolean {
        const { object, relation } = parseSubjectSet(subject);
        state.trace.push(`Resolve subject set ${subject}: does it contain ${user}?`);
        return evaluate(user, object, relation, state);
    }

    return { check };
};

type EvaluationState = {
    trace:      string[];
    evaluating: Set<VisitKey>;
    memo:       Map<VisitKey, boolean>;
};

export default makeGraphAuthorizer;
