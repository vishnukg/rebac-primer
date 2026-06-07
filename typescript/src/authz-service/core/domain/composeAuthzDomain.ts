// Composes the AuthZ service's four domain operations into the AuthzService port.
//
// A compose* (not a make*): it builds each operation via a make* factory, then
// bundles them — the same role composeDocuments plays on the documents side. The
// result is the boundary the HTTP adapter calls into; it does not know whether
// tuples live in memory or Postgres, or whether evaluation is in-process or via
// OpenFGA. (makeOpenFgaAuthzService implements the same AuthzService port itself.)

import makeCheck from "./makeCheck.ts";
import makeWriteTuples from "./makeWriteTuples.ts";
import makeDeleteTuples from "./makeDeleteTuples.ts";
import makeListTuples from "./makeListTuples.ts";
import type { TupleRepository, Evaluator } from "../ports/index.ts";
import type { AuthzService } from "./types.ts";

type ComposeAuthzDomainCfg = {
    repository: TupleRepository;
    evaluator:  Evaluator;
};

const composeAuthzDomain = ({
    repository,
    evaluator,
}: ComposeAuthzDomainCfg): AuthzService => {
    const check = makeCheck({ evaluator });
    const writeTuples = makeWriteTuples({ repository });
    const deleteTuples = makeDeleteTuples({ repository });
    const listTuples = makeListTuples({ repository });
    return { check, writeTuples, deleteTuples, listTuples };
};

export default composeAuthzDomain;
