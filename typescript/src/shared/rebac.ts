// Shared ReBAC types — the vocabulary that the authz service and any product
// service (documents, billing, etc.) both speak.
//
// In production this would be the published SDK package
// (@your-org/authz-sdk). Here it lives in src/shared/ as a local module.
//
// The authz service owns these concepts; product services import them
// to express check requests and write relationship tuples.

// ── Object types ──────────────────────────────────────────────────────────────

export type ObjectType = "user" | "team" | "workspace" | "document";

// A typed entity reference in "type:id" format.  TypeScript's template literal
// types enforce correct prefixes at compile time.
// Examples: "user:alice", "document:roadmap"
export type RebacObject<TType extends ObjectType = ObjectType> = `${TType}:${string}`;

// ── Relation names ─────────────────────────────────────────────────────────────

export type TeamRelation      = "admin" | "member";
export type WorkspaceRelation = "owner" | "editor" | "viewer";
export type DocumentRelation  =
    | "workspace"
    | "owner" | "editor" | "viewer"
    | "can_read" | "can_comment" | "can_edit" | "can_delete";

export type Relation = TeamRelation | WorkspaceRelation | DocumentRelation;

// ── Subjects ──────────────────────────────────────────────────────────────────

// A subject set: "team:platform#member" = everyone who holds `member` on `team:platform`.
export type TeamSubjectSet = `${RebacObject<"team">}#${TeamRelation}`;
export type SubjectSet     = TeamSubjectSet;
export type Subject        = RebacObject | SubjectSet;

// ── Tuple ─────────────────────────────────────────────────────────────────────

// One edge in the relationship graph.  Asserts "`user` has `relation` on `object`".
// Field named `user` to match OpenFGA's tuple API — it can be a concrete object
// or a subject set.
export type TupleKey = {
    object:   RebacObject;
    relation: Relation;
    user:     Subject;
};

// ── Check types ───────────────────────────────────────────────────────────────

export type CheckRequest = {
    user:     RebacObject<"user">;
    relation: Relation;
    object:   RebacObject;
};

export type CheckResult = {
    allowed: boolean;
    trace:   string[];
};

// ── Helper constructors ───────────────────────────────────────────────────────

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
    subject:  Subject,
): TupleKey => ({ object: objectId, relation, user: subject });

// ── Parse helpers ─────────────────────────────────────────────────────────────

export const parseObject = (value: string): { type: ObjectType; id: string } => {
    const [type, ...rest] = value.split(":");
    const id = rest.join(":");
    if (!isObjectType(type) || id.length === 0) {
        throw new Error(`Invalid ReBAC object: ${value}`);
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

export const isObjectOfType = <T extends ObjectType>(
    value: string,
    type: T,
): value is RebacObject<T> => {
    try { return parseObject(value).type === type; }
    catch { return false; }
};

// ── Private ───────────────────────────────────────────────────────────────────

const makeObject = <T extends ObjectType>(type: T, id: string): RebacObject<T> => {
    if (id.trim().length === 0) throw new Error(`${type} id cannot be empty`);
    return `${type}:${id}`;
};

const isObjectType = (v: string | undefined): v is ObjectType =>
    v === "user" || v === "team" || v === "workspace" || v === "document";

const isTeamRelation = (v: string | undefined): v is TeamRelation =>
    v === "admin" || v === "member";
