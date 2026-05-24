// Assembles the AuthZ service's four domain operations.
//
// This is the boundary the HTTP adapter calls into — it does not know whether
// tuples are stored in memory or in Postgres, or whether evaluation is done
// in-process or via OpenFGA.

import type { TupleRepository } from "../ports/tupleRepository.ts";
import type { AuthzService, TupleFilter } from "./types.ts";
import type { CheckRequest, CheckResult, TupleKey } from "../../../shared/rebac.ts";

type Evaluator = {
    evaluate: (request: CheckRequest) => CheckResult;
};

type AuthzDomainCfg = {
    repository: TupleRepository;
    evaluator:  Evaluator;
};

const makeAuthzDomain = ({ repository, evaluator }: AuthzDomainCfg): AuthzService => {
    const check = async (request: CheckRequest): Promise<CheckResult> =>
        evaluator.evaluate(request);

    const writeTuples = async (tuples: TupleKey[]): Promise<void> => {
        for (const t of tuples) repository.write(t);
    };

    const deleteTuples = async (tuples: TupleKey[]): Promise<void> => {
        for (const t of tuples) repository.delete(t);
    };

    const listTuples = async (filter?: TupleFilter): Promise<TupleKey[]> =>
        repository.findAll(filter);

    return { check, writeTuples, deleteTuples, listTuples };
};

export default makeAuthzDomain;
