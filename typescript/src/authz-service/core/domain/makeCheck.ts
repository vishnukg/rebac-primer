import type { Evaluator } from "../ports/index.ts";
import type { AuthzService } from "./types.ts";

// check() traverses the ReBAC graph via the evaluator. Thin today, but owning the
// domain's "check" operation gives it a home for caching, audit, etc. later.
const makeCheck = ({ evaluator }: { evaluator: Evaluator }): AuthzService["check"] => {
    return (request) => evaluator.evaluate(request);
};

export default makeCheck;
