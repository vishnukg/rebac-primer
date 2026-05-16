import { MemoryTupleStore } from "./memory-store.js";
import {
  type Authorizer,
  type CheckRequest,
  type CheckResult,
  type Relation,
  type RebacObject,
  type SubjectSet,
  isObjectOfType,
  isSubjectSet,
  parseObject,
  parseSubjectSet
} from "./types.js";

type VisitKey = `${RebacObject}#${Relation}`;

export class GraphAuthorizer implements Authorizer {
  constructor(private readonly store: MemoryTupleStore) {}

  async check(request: CheckRequest): Promise<CheckResult> {
    const trace: string[] = [
      `Check whether ${request.user} has ${request.relation} on ${request.object}`
    ];
    const allowed = this.hasRelation(request.user, request.object, request.relation, trace, new Set());
    trace.push(allowed ? "Result: allowed" : "Result: denied");
    return { allowed, trace };
  }

  private hasRelation(
    user: RebacObject<"user">,
    object: RebacObject,
    relation: Relation,
    trace: string[],
    visited: Set<VisitKey>
  ): boolean {
    const visitKey: VisitKey = `${object}#${relation}`;
    if (visited.has(visitKey)) {
      trace.push(`Already evaluated ${visitKey}; stop this branch`);
      return false;
    }
    visited.add(visitKey);

    if (this.hasDirectUserOrSubjectSet(user, object, relation, trace, visited)) {
      return true;
    }

    const { type } = parseObject(object);

    if (type === "team") {
      return this.expandTeam(user, object, relation, trace, visited);
    }

    if (type === "workspace") {
      return this.expandWorkspace(user, object, relation, trace, visited);
    }

    if (type === "document") {
      return this.expandDocument(user, object, relation, trace, visited);
    }

    return false;
  }

  private expandTeam(
    user: RebacObject<"user">,
    object: RebacObject,
    relation: Relation,
    trace: string[],
    visited: Set<VisitKey>
  ): boolean {
    if (relation === "admin") {
      trace.push("team.admin includes team.member");
      return this.hasRelation(user, object, "member", trace, visited);
    }

    return false;
  }

  private expandWorkspace(
    user: RebacObject<"user">,
    object: RebacObject,
    relation: Relation,
    trace: string[],
    visited: Set<VisitKey>
  ): boolean {
    if (relation === "editor") {
      trace.push("workspace.editor includes workspace.owner");
      return this.hasRelation(user, object, "owner", trace, visited);
    }

    if (relation === "viewer") {
      trace.push("workspace.viewer includes workspace.editor");
      return this.hasRelation(user, object, "editor", trace, visited);
    }

    return false;
  }

  private expandDocument(
    user: RebacObject<"user">,
    object: RebacObject,
    relation: Relation,
    trace: string[],
    visited: Set<VisitKey>
  ): boolean {
    const checks: Record<string, readonly Relation[]> = {
      can_read: ["viewer"],
      can_comment: ["viewer"],
      can_edit: ["editor"],
      can_delete: ["owner"],
      viewer: ["editor"],
      editor: ["owner"]
    };

    for (const impliedRelation of checks[relation] ?? []) {
      trace.push(`document.${relation} includes document.${impliedRelation}`);
      if (this.hasRelation(user, object, impliedRelation, trace, visited)) {
        return true;
      }
    }

    if (relation === "owner" || relation === "editor" || relation === "viewer") {
      const workspaceRelation = relation;
      for (const parent of this.store.findByObjectRelation(object, "workspace")) {
        trace.push(`document.${relation} can inherit workspace.${workspaceRelation} from ${parent.user}`);
        if (
          isObjectOfType(parent.user, "workspace") &&
          this.hasRelation(user, parent.user, workspaceRelation, trace, visited)
        ) {
          return true;
        }
      }
    }

    return false;
  }

  private hasDirectUserOrSubjectSet(
    user: RebacObject<"user">,
    object: RebacObject,
    relation: Relation,
    trace: string[],
    visited: Set<VisitKey>
  ): boolean {
    if (this.store.has(object, relation, user)) {
      trace.push(`Found direct tuple (${object}, ${relation}, ${user})`);
      return true;
    }

    for (const tupleKey of this.store.findByObjectRelation(object, relation)) {
      if (isSubjectSet(tupleKey.user) && this.subjectSetContains(user, tupleKey.user, trace, visited)) {
        trace.push(`Found subject-set tuple (${object}, ${relation}, ${tupleKey.user})`);
        return true;
      }
    }

    return false;
  }

  private subjectSetContains(
    user: RebacObject<"user">,
    subject: SubjectSet,
    trace: string[],
    visited: Set<VisitKey>
  ): boolean {
    const { object, relation } = parseSubjectSet(subject);
    trace.push(`Resolve subject set ${subject}: does it contain ${user}?`);
    return this.hasRelation(user, object, relation, trace, visited);
  }
}
