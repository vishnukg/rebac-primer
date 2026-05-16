import type { Relation, RebacObject, Subject, TupleKey } from "./types.js";

export class MemoryTupleStore {
  private readonly tuples = new Map<string, TupleKey>();

  constructor(seed: readonly TupleKey[] = []) {
    seed.forEach((tupleKey) => this.write(tupleKey));
  }

  write(tupleKey: TupleKey): void {
    this.tuples.set(keyFor(tupleKey), tupleKey);
  }

  delete(tupleKey: TupleKey): void {
    this.tuples.delete(keyFor(tupleKey));
  }

  has(object: RebacObject, relation: Relation, user: Subject): boolean {
    return this.tuples.has(keyFor({ object, relation, user }));
  }

  findByObjectRelation(object: RebacObject, relation: Relation): readonly TupleKey[] {
    return [...this.tuples.values()].filter(
      (tupleKey) => tupleKey.object === object && tupleKey.relation === relation
    );
  }

  all(): readonly TupleKey[] {
    return [...this.tuples.values()];
  }
}

function keyFor(tupleKey: TupleKey): string {
  return `${tupleKey.object}|${tupleKey.relation}|${tupleKey.user}`;
}
