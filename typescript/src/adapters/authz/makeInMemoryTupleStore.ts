import type { Relation, RebacObject, Subject, TupleKey, TupleStore } from "../../core/index.ts";

type InMemoryTupleStoreCfg = {
    seed?: TupleKey[];
};

const makeInMemoryTupleStore = ({ seed = [] }: InMemoryTupleStoreCfg = {}): TupleStore => {
    const tuples = new Map<string, TupleKey>();

    const write = (tupleKey: TupleKey): void => {
        tuples.set(keyFor(tupleKey), tupleKey);
    };

    const has = (object: RebacObject, relation: Relation, user: Subject): boolean =>
        tuples.has(keyFor({ object, relation, user }));

    const findByObjectRelation = (object: RebacObject, relation: Relation): TupleKey[] =>
        [...tuples.values()].filter(
            t => t.object === object && t.relation === relation,
        );

    seed.forEach(write);

    return { has, findByObjectRelation };
};

const keyFor = (t: TupleKey): string => `${t.object}|${t.relation}|${t.user}`;

export default makeInMemoryTupleStore;
