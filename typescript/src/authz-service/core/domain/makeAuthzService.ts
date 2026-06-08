// The AuthZ domain — one factory that closes over its driven ports (the tuple
// repository and the evaluator) and exposes the four operations as methods. This
// is the module pattern at its core: dependencies hidden in the closure,
// behaviour exposed as a noun (AuthzService) whose methods are verbs (check,
// writeTuples, deleteTuples, listTuples).
//
// One factory, one noun: the domain IS the AuthzService port, so its operations
// live inside it as methods rather than as four standalone make* factories that
// then have to be re-bundled. It calls no other factory, so it is a leaf (a
// make*, not a compose*) — mirrors makeRestaurant in the ModulePattern repo and
// makeDocuments on the documents side.
//
// This is the boundary the HTTP adapter calls into; it does not know whether
// tuples live in memory or Postgres. (makeOpenFgaAuthzService is a second adapter
// implementing the same AuthzService port over a real OpenFGA server — the
// in-process / OpenFGA choice is made in compose.ts, exactly like makeInMemoryDb
// vs makeDynamoDb in the ModulePattern repo.)

import type { TupleRepository, Evaluator } from "../ports/index.ts";
import type { AuthzService } from "./types.ts";

type MakeAuthzServiceCfg = {
    repository: TupleRepository;
    evaluator:  Evaluator;
};

const makeAuthzService = ({ repository, evaluator }: MakeAuthzServiceCfg): AuthzService => {
    // check() traverses the ReBAC graph via the evaluator. Thin today, but owning
    // the domain's "check" operation gives it a home for caching, audit, etc. later.
    const check: AuthzService["check"] = (request) => evaluator.evaluate(request);

    const writeTuples: AuthzService["writeTuples"] = async (tuples) => {
        for (const t of tuples) repository.write(t);
    };

    const deleteTuples: AuthzService["deleteTuples"] = async (tuples) => {
        for (const t of tuples) repository.delete(t);
    };

    // Pure pass-through to the repository today — reading tuples adds no domain
    // logic. Owning the operation anyway gives one place to add filtering/paging/
    // audit later.
    const listTuples: AuthzService["listTuples"] = async (filter) => repository.findAll(filter);

    return { check, writeTuples, deleteTuples, listTuples };
};

export default makeAuthzService;
