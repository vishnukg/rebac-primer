export type ObjectType = "user" | "team" | "workspace" | "document";

export type RebacObject<TType extends ObjectType = ObjectType> = `${TType}:${string}`;

export type TeamRelation = "admin" | "member";
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
export type TeamSubjectSet = `${RebacObject<"team">}#${TeamRelation}`;
export type SubjectSet = TeamSubjectSet;
export type Subject = RebacObject | SubjectSet;

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
export type WriteTuplesFn = (tuples: TupleKey[]) => Promise<void>;

export interface Authorizer {
  check: CheckFn;
}

export interface TupleStore {
  write:                (tupleKey: TupleKey) => void;
  delete:               (tupleKey: TupleKey) => void;
  has:                  (object: RebacObject, relation: Relation, user: Subject) => boolean;
  findByObjectRelation: (object: RebacObject, relation: Relation) => TupleKey[];
  all:                  () => TupleKey[];
}

export const user = (id: string): RebacObject<"user"> => object("user", id);
export const team = (id: string): RebacObject<"team"> => object("team", id);
export const workspace = (id: string): RebacObject<"workspace"> => object("workspace", id);
export const document = (id: string): RebacObject<"document"> => object("document", id);

export const subjectSet = (objectId: RebacObject<"team">, relation: TeamRelation): TeamSubjectSet =>
  `${objectId}#${relation}`;

export const tuple = (objectId: RebacObject, relation: Relation, subject: Subject): TupleKey => ({
  object: objectId,
  relation,
  user: subject,
});

export const parseObject = (value: string): { type: ObjectType; id: string } => {
  const [type, ...idParts] = value.split(":");
  const id = idParts.join(":");
  if (!isObjectType(type) || id.length === 0) {
    throw new Error(`Invalid ReBAC object id: ${value}`);
  }
  return { type, id };
};

export const parseSubjectSet = (value: string): { object: RebacObject<"team">; relation: TeamRelation } => {
  const [objectId, relation] = value.split("#");
  if (!objectId || !isObjectOfType(objectId, "team") || !isTeamRelation(relation)) {
    throw new Error(`Invalid subject set: ${value}`);
  }
  return { object: objectId, relation };
};

export const isSubjectSet = (subject: Subject): subject is SubjectSet => subject.includes("#");

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

const object = <TType extends ObjectType>(type: TType, id: string): RebacObject<TType> => {
  if (id.trim().length === 0) {
    throw new Error(`${type} id cannot be be empty`);
  }
  return `${type}:${id}`;
};

const isObjectType = (value: string | undefined): value is ObjectType =>
  value === "user" || value === "team" || value === "workspace" || value === "document";

const isTeamRelation = (value: string | undefined): value is TeamRelation =>
  value === "admin" || value === "member";
