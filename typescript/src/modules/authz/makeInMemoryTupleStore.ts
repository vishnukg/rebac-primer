import type { Relation, RebacObject, Subject, TupleKey, TupleStore } from "./types.ts";

type InMemoryTupleStoreCfg = {
  seed?: TupleKey[];
};

const makeInMemoryTupleStore = ({ seed = [] }: InMemoryTupleStoreCfg = {}): TupleStore => {
  const tuples = new Map<string, TupleKey>();

  const write = (tupleKey: TupleKey): void => {
    tuples.set(keyFor(tupleKey), tupleKey);
  };

  const remove = (tupleKey: TupleKey): void => {
    tuples.delete(keyFor(tupleKey));
  };

  const has = (object: RebacObject, relation: Relation, user: Subject): boolean =>
    tuples.has(keyFor({ object, relation, user }));

  const findByObjectRelation = (object: RebacObject, relation: Relation): TupleKey[] =>
    [...tuples.values()].filter(tupleKey => tupleKey.object === object && tupleKey.relation === relation);

  const all = (): TupleKey[] => [...tuples.values()];

  seed.forEach(write);

  return { write, delete: remove, has, findByObjectRelation, all };
};

const keyFor = (tupleKey: TupleKey): string =>
  `${tupleKey.object}|${tupleKey.relation}|${tupleKey.user}`;

export default makeInMemoryTupleStore;
