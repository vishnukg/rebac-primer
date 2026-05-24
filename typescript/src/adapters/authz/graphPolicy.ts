import type { ObjectType, Relation } from "../../core/index.ts";

export type GraphResourceType = Exclude<ObjectType, "user">;

export type ParentRelationPolicy = {
    parentObjectType:   GraphResourceType;
    parentTupleRelation: Relation;
    inheritedRelations:  ReadonlySet<Relation>;
};

export type ObjectRelationPolicy = {
    objectType:               GraphResourceType;
    sameObjectImpliedBy:      ReadonlyMap<Relation, readonly Relation[]>;
    parentRelationInheritance?: ParentRelationPolicy;
};

export interface AuthorizationPolicy {
    policyFor: (objectType: ObjectType) => ObjectRelationPolicy | undefined;
}

const mapRelations = (
    entries: readonly (readonly [Relation, readonly Relation[]])[],
): ReadonlyMap<Relation, readonly Relation[]> => new Map(entries);

// These rules mirror model.ts in data structures that match how the evaluator
// reads them: first same-object rewrites, then optional parent traversal.
const policies = new Map<GraphResourceType, ObjectRelationPolicy>([
    [
        "team",
        {
            objectType: "team",
            sameObjectImpliedBy: mapRelations([["member", ["admin"]]]),
        },
    ],
    [
        "workspace",
        {
            objectType: "workspace",
            sameObjectImpliedBy: mapRelations([
                ["viewer", ["editor"]],
                ["editor", ["owner"]],
            ]),
        },
    ],
    [
        "document",
        {
            objectType: "document",
            sameObjectImpliedBy: mapRelations([
                ["can_read", ["viewer"]],
                ["can_comment", ["viewer"]],
                ["can_edit", ["editor"]],
                ["can_delete", ["owner"]],
                ["viewer", ["editor"]],
                ["editor", ["owner"]],
            ]),
            parentRelationInheritance: {
                parentObjectType:    "workspace",
                parentTupleRelation: "workspace",
                inheritedRelations:  new Set(["owner", "editor", "viewer"]),
            },
        },
    ],
]);

export const staticAuthorizationPolicy: AuthorizationPolicy = {
    policyFor: objectType =>
        objectType === "user" ? undefined : policies.get(objectType),
};
