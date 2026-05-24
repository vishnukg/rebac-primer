// AuthZ service domain types.
//
// The domain owns three operations:
//   check       — traverse the graph, return allowed/denied
//   writeTuples — add relationship assertions
//   deleteTuples— remove relationship assertions
//   listTuples  — read back stored tuples (for audit/debugging)

import type { CheckRequest, CheckResult, Relation, RebacObject, TupleKey } from "../../../shared/rebac.ts";

export type { CheckRequest, CheckResult, TupleKey };

// ── Tuple filter ──────────────────────────────────────────────────────────────

// Narrows which tuples listTuples returns.
export type TupleFilter = {
    object?:   RebacObject;
    relation?: Relation;
};

// ── Driving port ──────────────────────────────────────────────────────────────

// This is what the HTTP adapter calls into.  It is also what an SDK would wrap.
export type AuthzService = {
    check:        (request: CheckRequest) => Promise<CheckResult>;
    writeTuples:  (tuples: TupleKey[]) => Promise<void>;
    deleteTuples: (tuples: TupleKey[]) => Promise<void>;
    listTuples:   (filter?: TupleFilter) => Promise<TupleKey[]>;
};

// ── Domain errors ─────────────────────────────────────────────────────────────

export type TupleValidationError = Error & { readonly name: "TupleValidationError" };
export const TupleValidationError = (message: string): TupleValidationError =>
    Object.assign(new Error(message), { name: "TupleValidationError" as const });
export const isTupleValidationError = (e: unknown): e is TupleValidationError =>
    e instanceof Error && e.name === "TupleValidationError";
