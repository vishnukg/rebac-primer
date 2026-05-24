// Driven port — the strategy the authz domain uses to evaluate a check request.
//
// The domain calls evaluate(); the adapter decides how:
//   makeGraphEvaluator  — in-process ReBAC graph traversal (this project)
//   OpenFGA SDK         — remote evaluation via OpenFGA API

import type { CheckRequest, CheckResult } from "../../../shared/rebac.ts";

export interface Evaluator {
    evaluate: (request: CheckRequest) => Promise<CheckResult>;
}
