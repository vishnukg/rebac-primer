// This file is the public authz boundary for the core application.
// It deliberately contains only stable ReBAC concepts and ports. Concrete
// implementations live in adapters/authz.

export type ObjectType = "user" | "team" | "workspace" | "document";

// A ReBAC object is an OpenFGA-style typed id: "type:id".
// Examples: "user:alice", "document:roadmap".
export type RebacObject<TType extends ObjectType = ObjectType> = `${TType}:${string}`;

export type TeamRelation      = "admin" | "member";
export type WorkspaceRelation = "owner" | "editor" | "viewer";
export type DocumentRelation  =
    | "workspace"
    | "owner"
    | "editor"
    | "viewer"
    | "can_read"
    | "can_comment"
    | "can_edit"
    | "can_delete";

export type Relation = TeamRelation | WorkspaceRelation | DocumentRelation;

// A subject set references everyone who holds a relation on an object.
// "team:platform#member" means "everyone who is a member of team:platform".
export type TeamSubjectSet = `${RebacObject<"team">}#${TeamRelation}`;
export type SubjectSet     = TeamSubjectSet;
export type Subject        = RebacObject | SubjectSet;

// A relationship tuple asserts that `user` has `relation` on `object`.
// The field is named `user` to match OpenFGA's tuple API. It can be either a
// concrete user/object or a subject set such as "team:platform#member".
export type TupleKey = {
    object:   RebacObject;
    relation: Relation;
    user:     Subject;
};

export type CheckRequest = {
    user:     RebacObject<"user">;
    relation: Relation;
    object:   RebacObject;
};

export type CheckResult = {
    allowed: boolean;
    trace:   string[];
};

export type CheckFn = (request: CheckRequest) => Promise<CheckResult>;

// Driven port: the domain calls this to ask "is this action allowed?"
// Adapters decide how to answer: in-process graph traversal, OpenFGA, etc.
export interface Authorizer {
    check: CheckFn;
}

// Driven port used by graph-based authorizers to read stored relationships.
// Production adapters can back this with SQL, OpenFGA exports, or another store.
export interface TupleStore {
    has:                  (object: RebacObject, relation: Relation, user: Subject) => boolean;
    findByObjectRelation: (object: RebacObject, relation: Relation) => TupleKey[];
}

export const user      = (id: string): RebacObject<"user">      => makeObject("user", id);
export const team      = (id: string): RebacObject<"team">      => makeObject("team", id);
export const workspace = (id: string): RebacObject<"workspace"> => makeObject("workspace", id);
export const document  = (id: string): RebacObject<"document">  => makeObject("document", id);

export const subjectSet = (
    objectId: RebacObject<"team">,
    relation: TeamRelation,
): TeamSubjectSet => `${objectId}#${relation}`;

export const tuple = (
    objectId: RebacObject,
    relation: Relation,
    subject: Subject,
): TupleKey => ({ object: objectId, relation, user: subject });

export const parseObject = (value: string): { type: ObjectType; id: string } => {
    const [type, ...idParts] = value.split(":");
    const id = idParts.join(":");
    if (!isObjectType(type) || id.length === 0) {
        throw new Error(`Invalid ReBAC object id: ${value}`);
    }
    return { type, id };
};

export const parseSubjectSet = (
    value: string,
): { object: RebacObject<"team">; relation: TeamRelation } => {
    const [objectId, relation] = value.split("#");
    if (!objectId || !isObjectOfType(objectId, "team") || !isTeamRelation(relation)) {
        throw new Error(`Invalid subject set: ${value}`);
    }
    return { object: objectId, relation };
};

export const isSubjectSet = (subject: Subject): subject is SubjectSet =>
    subject.includes("#");

export const isObjectOfType = <TType extends ObjectType>(
    value: string,
    type: TType,
): value is RebacObject<TType> => {
    try {
        return parseObject(value).type === type;
    } catch {
        return false;
    }
};

// ── Internal ──────────────────────────────────────────────────────────────────

const makeObject = <TType extends ObjectType>(type: TType, id: string): RebacObject<TType> => {
    if (id.trim().length === 0) throw new Error(`${type} id cannot be empty`);
    return `${type}:${id}`;
};

const isObjectType = (value: string | undefined): value is ObjectType =>
    value === "user" || value === "team" || value === "workspace" || value === "document";

export const isTeamRelation = (value: string | undefined): value is TeamRelation =>
    value === "admin" || value === "member";
