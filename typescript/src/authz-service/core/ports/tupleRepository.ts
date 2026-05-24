// Driven port — what the authz domain needs from a persistence layer.
//
// The graph evaluator reads from this; the write operations mutate it.
// Adapters decide the storage backend: in-memory, Postgres, OpenFGA exports, etc.

import type { Relation, RebacObject, Subject, TupleKey } from "../../../shared/rebac.ts";

export interface TupleRepository {
    // Returns true if the exact (object, relation, user) tuple exists.
    has: (object: RebacObject, relation: Relation, user: Subject) => boolean;

    // Returns all tuples matching (object, relation).
    // Used during graph traversal and list operations.
    findByObjectRelation: (object: RebacObject, relation: Relation) => TupleKey[];

    // Returns all tuples, optionally filtered.
    findAll: (filter?: { object?: RebacObject; relation?: Relation }) => TupleKey[];

    // Adds a tuple (idempotent).
    write: (tuple: TupleKey) => void;

    // Removes a tuple.  No-op if it does not exist.
    delete: (tuple: TupleKey) => void;
}
