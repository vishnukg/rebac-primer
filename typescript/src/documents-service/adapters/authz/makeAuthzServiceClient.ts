// HTTP adapter — calls the AuthZ service over HTTP to check permissions and
// write tuples.  Implements the AuthzClient driven port.
//
// This is what separates "the documents service talking to its own in-process
// graph evaluator" (the old design) from "the documents service talking to a
// dedicated authz service" (the new design).  The domain code is identical;
// only this adapter changes.

import type { AuthzClient } from "../../core/ports/authzClient.ts";
import type { CheckRequest, CheckResult, TupleKey } from "../../../shared/rebac.ts";

type Fetcher = (url: URL, init?: RequestInit) => Promise<Response>;

type AuthzServiceClientCfg = {
    baseUrl: string;
    fetcher?: Fetcher;
};

const makeAuthzServiceClient = ({
    baseUrl,
    fetcher = fetch,
}: AuthzServiceClientCfg): AuthzClient => {
    const post = async (path: string, body: unknown): Promise<unknown> => {
        let response: Response;
        try {
            response = await fetcher(new URL(path, baseUrl), {
                method:  "POST",
                headers: { "content-type": "application/json" },
                body:    JSON.stringify(body),
            });
        } catch (error) {
            // Network-level failure (DNS, connection refused, timeout). Preserve
            // the original error via `cause` (ES2022) so the reason is not lost.
            throw new Error(`AuthZ service unreachable at ${baseUrl}${path}`, { cause: error });
        }
        const json: unknown = await response.json();
        if (!response.ok) {
            const msg = isErrorBody(json) ? json.error : response.statusText;
            throw new Error(`AuthZ service error: ${msg}`);
        }
        return json;
    };

    const check = async (request: CheckRequest): Promise<CheckResult> => {
        const result = await post("/check", {
            user:     request.user,
            relation: request.relation,
            object:   request.object,
        });
        if (!isCheckResult(result)) {
            throw new Error("AuthZ service returned unexpected response from /check");
        }
        return result;
    };

    const writeTuples = async (tuples: TupleKey[]): Promise<void> => {
        await post("/tuples", { tuples });
    };

    return { check, writeTuples };
};

const isCheckResult = (v: unknown): v is CheckResult =>
    typeof v === "object" && v !== null &&
    "allowed" in v && typeof (v as Record<string, unknown>).allowed === "boolean";

const isErrorBody = (v: unknown): v is { error: string } =>
    typeof v === "object" && v !== null &&
    "error" in v && typeof (v as Record<string, unknown>).error === "string";

export default makeAuthzServiceClient;
