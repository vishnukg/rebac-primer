import type {
  Authorizer,
  CheckRequest,
  CheckResult,
  RebacObject,
  Relation,
  SubjectSet,
  TupleStore,
} from "./types.ts";
import { isObjectOfType, isSubjectSet, parseObject, parseSubjectSet } from "./types.ts";

type GraphAuthorizerCfg = {
  tupleStore: Pick<TupleStore, "has" | "findByObjectRelation">;
};

type VisitKey = `${RebacObject}#${Relation}`;

const makeGraphAuthorizer = ({ tupleStore }: GraphAuthorizerCfg): Authorizer => {
  const check = async (request: CheckRequest): Promise<CheckResult> => {
    const trace = [`Check whether ${request.user} has ${request.relation} on ${request.object}`];
    const allowed = hasRelation(request.user, request.object, request.relation, trace, new Set());
    trace.push(allowed ? "Result: allowed" : "Result: denied");
    return { allowed, trace };
  };

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

    if (hasDirectUserOrSubjectSet(user, object, relation, trace, visited)) return true;

    const { type } = parseObject(object);
    if (type === "team") return expandTeam(user, object, relation, trace, visited);
    if (type === "workspace") return expandWorkspace(user, object, relation, trace, visited);
    if (type === "document") return expandDocument(user, object, relation, trace, visited);
    return false;
  };

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

  const expandDocument = (
    user: RebacObject<"user">,
    object: RebacObject,
    relation: Relation,
    trace: string[],
    visited: Set<VisitKey>,
  ): boolean => {
    const implied: Partial<Record<Relation, Relation[]>> = {
      can_read:    ["viewer"],
      can_comment: ["viewer"],
      can_edit:    ["editor"],
      can_delete:  ["owner"],
      viewer:      ["editor"],
      editor:      ["owner"],
    };

    for (const impliedRelation of implied[relation] ?? []) {
      trace.push(`document.${relation} includes document.${impliedRelation}`);
      if (hasRelation(user, object, impliedRelation, trace, visited)) return true;
    }

    if (relation === "owner" || relation === "editor" || relation === "viewer") {
      for (const parent of tupleStore.findByObjectRelation(object, "workspace")) {
        trace.push(`document.${relation} can inherit workspace.${relation} from ${parent.user}`);
        if (isObjectOfType(parent.user, "workspace") && hasRelation(user, parent.user, relation, trace, visited)) {
          return true;
        }
      }
    }

    return false;
  };

  const hasDirectUserOrSubjectSet = (
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

    for (const tupleKey of tupleStore.findByObjectRelation(object, relation)) {
      if (isSubjectSet(tupleKey.user) && subjectSetContains(user, tupleKey.user, trace, visited)) {
        trace.push(`Found subject-set tuple (${object}, ${relation}, ${tupleKey.user})`);
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
