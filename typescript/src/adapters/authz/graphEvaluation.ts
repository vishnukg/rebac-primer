import {
    isObjectOfType,
    isSubjectSet,
    parseObject,
    parseSubjectSet,
} from "../../core/index.ts";
import type {
    CheckRequest,
    CheckResult,
    RebacObject,
    Relation,
    SubjectSet,
    TupleKey,
    TupleStore,
} from "../../core/index.ts";
import type { AuthorizationPolicy, ObjectRelationPolicy } from "./graphPolicy.ts";

type TupleReader = Pick<TupleStore, "has" | "findByObjectRelation">;

export interface RelationshipReader {
    hasDirectRelationship: (relationship: Relationship) => boolean;
    findRelationships: (object: RebacObject, relation: Relation) => TupleKey[];
}

export interface PermissionEvaluator {
    check: (request: CheckRequest) => CheckResult;
}

export type Relationship = {
    object:   RebacObject;
    relation: Relation;
    subject:  RebacObject<"user">;
};

export type PermissionEvaluatorCfg = {
    relationships: RelationshipReader;
    policy:        AuthorizationPolicy;
};

type VisitKey = `${RebacObject}#${Relation}`;

type TraversalState = {
    readonly trace:      string[];
    readonly evaluating: Set<VisitKey>;
    readonly memo:       Map<VisitKey, boolean>;
};

type RelationCheck = {
    user:     RebacObject<"user">;
    object:   RebacObject;
    relation: Relation;
};

type RelationCheckContext = RelationCheck & {
    relationships: RelationshipReader;
    policy:        AuthorizationPolicy;
    state:         TraversalState;
};

type PolicyCheckContext = RelationCheckContext & {
    objectPolicy: ObjectRelationPolicy;
};

export const makeTupleStoreRelationshipReader = (
    tupleStore: TupleReader,
): RelationshipReader => ({
    hasDirectRelationship: ({ object, relation, subject }) =>
        tupleStore.has(object, relation, subject),
    findRelationships: (object, relation) =>
        tupleStore.findByObjectRelation(object, relation),
});

export const makeGraphPermissionEvaluator = ({
    relationships,
    policy,
}: PermissionEvaluatorCfg): PermissionEvaluator => ({
    check: request => {
        const state: TraversalState = {
            trace: [
                `Check whether ${request.user} has ${request.relation} on ${request.object}`,
            ],
            evaluating: new Set(),
            memo:       new Map(),
        };

        const allowed = evaluateRelation({
            user: request.user,
            object: request.object,
            relation: request.relation,
            relationships,
            policy,
            state,
        });
        state.trace.push(allowed ? "Result: allowed" : "Result: denied");

        return { allowed, trace: state.trace };
    },
});

const evaluateRelation = (context: RelationCheckContext): boolean => {
    const visitKey = toVisitKey(context);
    const cached = context.state.memo.get(visitKey);
    if (cached !== undefined) return cached;

    if (context.state.evaluating.has(visitKey)) {
        context.state.trace.push(`Already evaluating ${visitKey}; stop this cycle`);
        return false;
    }

    context.state.evaluating.add(visitKey);
    const allowed = evaluateRelationWithoutMemo(context);
    context.state.evaluating.delete(visitKey);
    context.state.memo.set(visitKey, allowed);
    return allowed;
};

const evaluateRelationWithoutMemo = (context: RelationCheckContext): boolean => {
    if (hasStoredRelationship(context)) return true;

    const objectType = parseObject(context.object).type;
    const objectPolicy = context.policy.policyFor(objectType);
    if (!objectPolicy) return false;

    return (
        hasSameObjectInheritedRelation({ ...context, objectPolicy }) ||
        hasParentInheritedRelation({ ...context, objectPolicy })
    );
};

const hasStoredRelationship = (context: RelationCheckContext): boolean => {
    if (
        context.relationships.hasDirectRelationship({
            object: context.object,
            relation: context.relation,
            subject: context.user,
        })
    ) {
        context.state.trace.push(
            `Found direct tuple (${context.object}, ${context.relation}, ${context.user})`,
        );
        return true;
    }

    for (const tupleKey of context.relationships.findRelationships(
        context.object,
        context.relation,
    )) {
        if (
            isSubjectSet(tupleKey.user) &&
            subjectSetContainsUser(context, tupleKey.user)
        ) {
            context.state.trace.push(
                `Found subject-set tuple (${context.object}, ${context.relation}, ${tupleKey.user})`,
            );
            return true;
        }
    }

    return false;
};

const hasSameObjectInheritedRelation = (context: PolicyCheckContext): boolean => {
    const impliedBy = context.objectPolicy.sameObjectImpliedBy.get(context.relation) ?? [];
    for (const strongerRelation of impliedBy) {
        context.state.trace.push(
            `${context.object} ${context.relation} includes ${strongerRelation}`,
        );
        if (evaluateRelation({ ...context, relation: strongerRelation })) {
            return true;
        }
    }

    return false;
};

const hasParentInheritedRelation = (context: PolicyCheckContext): boolean => {
    const parentPolicy = context.objectPolicy.parentRelationInheritance;
    if (!parentPolicy?.inheritedRelations.has(context.relation)) return false;

    for (const tupleKey of context.relationships.findRelationships(
        context.object,
        parentPolicy.parentTupleRelation,
    )) {
        context.state.trace.push(
            `${context.object} ${context.relation} can inherit ${context.relation} from ${tupleKey.user}`,
        );
        if (
            isObjectOfType(tupleKey.user, parentPolicy.parentObjectType) &&
            evaluateRelation({
                ...context,
                object: tupleKey.user,
            })
        ) {
            return true;
        }
    }

    return false;
};

const subjectSetContainsUser = (
    context: RelationCheckContext,
    subjectSet: SubjectSet,
): boolean => {
    const { object, relation } = parseSubjectSet(subjectSet);
    context.state.trace.push(
        `Resolve subject set ${subjectSet}: does it contain ${context.user}?`,
    );
    return evaluateRelation({ ...context, object, relation });
};

const toVisitKey = ({ object, relation }: RelationCheck): VisitKey =>
    `${object}#${relation}`;
