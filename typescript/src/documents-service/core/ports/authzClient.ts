// Driven port — what the documents domain needs from the authz service.
//
// The documents domain calls this port to:
//   1. check  — ask "can this actor do X on this object?"
//   2. writeTuples — register new relationships (called on document creation)
//
// In tests this is satisfied by a fake in-memory implementation.
// In production it is satisfied by makeAuthzServiceClient, which calls the
// AuthZ service over HTTP.

import type { CheckRequest, CheckResult, TupleKey } from "../../../shared/rebac.ts";

export type { CheckRequest, CheckResult, TupleKey };

export type AuthzClient = {
    check:       (request: CheckRequest) => Promise<CheckResult>;
    writeTuples: (tuples: TupleKey[]) => Promise<void>;
};
