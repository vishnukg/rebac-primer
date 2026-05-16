export type ObjectType = "user" | "team" | "workspace" | "document";

export type RebacObject<TType extends ObjectType = ObjectType> =
  `${TType}:${string}`;

export type SubjectSet = `${Exclude<ObjectType, "user">}:${string}#${string}`;

export type Subject = RebacObject | SubjectSet;

export type TeamRelation = "member" | "admin";
export type WorkspaceRelation = "owner" | "editor" | "viewer";
export type DocumentRelation =
  | "workspace"
  | "owner"
  | "editor"
  | "viewer"
  | "can_read"
  | "can_comment"
  | "can_edit"
  | "can_delete";

export type Relation = TeamRelation | WorkspaceRelation | DocumentRelation;

export type TupleKey = Readonly<{
  user: Subject;
  relation: Relation;
  object: RebacObject;
}>;

export type CheckRequest = Readonly<{
  user: RebacObject<"user">;
  relation: Relation;
  object: RebacObject;
}>;

export type CheckResult = Readonly<{
  allowed: boolean;
  trace: readonly string[];
}>;

export interface Authorizer {
  check(request: CheckRequest): Promise<CheckResult>;
}

export function user(id: string): RebacObject<"user"> {
  return object("user", id);
}

export function team(id: string): RebacObject<"team"> {
  return object("team", id);
}

export function workspace(id: string): RebacObject<"workspace"> {
  return object("workspace", id);
}

export function document(id: string): RebacObject<"document"> {
  return object("document", id);
}

export function subjectSet(objectId: RebacObject<"team">, relation: TeamRelation): SubjectSet {
  return `${objectId}#${relation}`;
}

export function tuple(objectId: RebacObject, relation: Relation, subject: Subject): TupleKey {
  return { object: objectId, relation, user: subject };
}

export function parseObject(value: RebacObject): { type: ObjectType; id: string } {
  const [type, ...idParts] = value.split(":");
  if (!isObjectType(type) || idParts.length === 0 || idParts.join(":").length === 0) {
    throw new Error(`Invalid OpenFGA object id: ${value}`);
  }

  return { type, id: idParts.join(":") };
}

export function parseSubjectSet(value: SubjectSet): {
  object: RebacObject;
  relation: Relation;
} {
  const [objectId, relation] = value.split("#");
  if (!objectId || !relation) {
    throw new Error(`Invalid subject set: ${value}`);
  }

  return { object: objectId as RebacObject, relation: relation as Relation };
}

function object<TType extends ObjectType>(type: TType, id: string): RebacObject<TType> {
  if (id.trim().length === 0) {
    throw new Error(`${type} id cannot be empty`);
  }

  return `${type}:${id}`;
}

function isObjectType(value: string | undefined): value is ObjectType {
  return value === "user" || value === "team" || value === "workspace" || value === "document";
}
