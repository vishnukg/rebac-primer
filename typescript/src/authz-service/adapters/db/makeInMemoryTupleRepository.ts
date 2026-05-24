// In-memory TupleRepository — for development and tests.
//
// Backed by a Map keyed on "object|relation|user".  A production adapter
// would use a relational DB with indexes on (object, relation).

import type { TupleFilter, TupleRepository } from "../../core/ports/index.ts";
import type { Relation, RebacObject, Subject, TupleKey } from "../../../shared/rebac.ts";

type InMemoryTupleRepositoryCfg = {
    seed?: TupleKey[];
};

const makeInMemoryTupleRepository = ({ seed = [] }: InMemoryTupleRepositoryCfg = {}): TupleRepository => {
    const store = new Map<string, TupleKey>();

    const keyFor = (t: TupleKey): string => `${t.object}|${t.relation}|${t.user}`;

    const write = (t: TupleKey): void => { store.set(keyFor(t), t); };
    seed.forEach(write);

    const has = (object: RebacObject, relation: Relation, user: Subject): boolean =>
        store.has(keyFor({ object, relation, user }));

    const findByObjectRelation = (object: RebacObject, relation: Relation): TupleKey[] =>
        [...store.values()].filter(t => t.object === object && t.relation === relation);

    const findAll = (filter?: TupleFilter): TupleKey[] => {
        if (!filter) return [...store.values()];
        return [...store.values()].filter(t =>
            (!filter.object   || t.object   === filter.object) &&
            (!filter.relation || t.relation === filter.relation),
        );
    };

    const deleteFn = (t: TupleKey): void => { store.delete(keyFor(t)); };

    return { has, findByObjectRelation, findAll, write, delete: deleteFn };
};

export default makeInMemoryTupleRepository;
