// HTTP adapter for the AuthZ service.
//
// Routes:
//   GET  /health
//   POST /check      { user, relation, object }        → { allowed, trace }
//   POST /tuples     { tuples: [{object,relation,user}] } → { written }
//   DELETE /tuples   { tuples: [{object,relation,user}] } → { deleted }
//   GET  /tuples     ?object=...&relation=...           → { tuples }
//
// Product services call POST /check to ask "can this user do that?".
// Product services call POST /tuples when relationships change (e.g. a new
// document is created, or a user is added to a team).

import type { AuthzService } from "../../core/index.ts";
import { isTupleValidationError } from "../../core/index.ts";
import { isJsonObject, stringField, optionalStringField } from "./json.ts";
import type { Relation, RebacObject, Subject, TupleKey } from "../../../shared/rebac.ts";
import type { TupleFilter } from "../../core/ports/index.ts";

export type HttpRequest = {
    method:        string;
    path:          string;
    query:         URLSearchParams;
    body?:         unknown;
};

export type HttpResponse = {
    statusCode: number;
    body:       Record<string, unknown>;
};

export type AuthzHttpHandler = (request: HttpRequest) => Promise<HttpResponse>;

const makeAuthzHttpHandler = (authz: AuthzService): AuthzHttpHandler => {
    const handle: AuthzHttpHandler = async request => {
        try {
            // ── Health ────────────────────────────────────────────────────────
            if (request.method === "GET" && request.path === "/health") {
                return json(200, { ok: true });
            }

            // ── Check permission ──────────────────────────────────────────────
            if (request.method === "POST" && request.path === "/check") {
                const body   = requireBody(request.body);
                const result = await authz.check({
                    user:     stringField(body, "user")     as RebacObject<"user">,
                    relation: stringField(body, "relation") as Relation,
                    object:   stringField(body, "object")   as RebacObject,
                });
                return json(200, { allowed: result.allowed, trace: result.trace });
            }

            // ── Write tuples ──────────────────────────────────────────────────
            if (request.method === "POST" && request.path === "/tuples") {
                const body   = requireBody(request.body);
                const tuples = parseTuples(body);
                await authz.writeTuples(tuples);
                return json(200, { written: tuples.length });
            }

            // ── Delete tuples ─────────────────────────────────────────────────
            if (request.method === "DELETE" && request.path === "/tuples") {
                const body   = requireBody(request.body);
                const tuples = parseTuples(body);
                await authz.deleteTuples(tuples);
                return json(200, { deleted: tuples.length });
            }

            // ── List tuples ───────────────────────────────────────────────────
            if (request.method === "GET" && request.path === "/tuples") {
                const object   = optionalStringField(request.query, "object");
                const relation = optionalStringField(request.query, "relation");
                const filter: TupleFilter = {};
                if (object   !== undefined) filter.object   = object   as RebacObject;
                if (relation !== undefined) filter.relation = relation as Relation;
                const tuples   = await authz.listTuples(filter);
                return json(200, { tuples });
            }

            return json(404, { error: "Route not found" });
        } catch (error) {
            return toErrorResponse(error);
        }
    };

    return handle;
};

// ── Helpers ───────────────────────────────────────────────────────────────────

const requireBody = (body: unknown): Record<string, unknown> => {
    if (!isJsonObject(body)) throw new Error("Request body must be a JSON object");
    return body;
};

const parseTuples = (body: Record<string, unknown>): TupleKey[] => {
    const raw = body.tuples;
    if (!Array.isArray(raw)) throw new Error("Field 'tuples' must be an array");
    return raw.map((item, i) => {
        if (!isJsonObject(item)) throw new Error(`tuples[${i}] must be an object`);
        return {
            object:   stringField(item, "object")   as RebacObject,
            relation: stringField(item, "relation") as Relation,
            user:     stringField(item, "user")     as Subject,
        };
    });
};

const toErrorResponse = (error: unknown): HttpResponse => {
    if (isTupleValidationError(error)) return json(422, { error: error.message });
    const message = error instanceof Error ? error.message : "Unknown error";
    return json(400, { error: message });
};

const json = (statusCode: number, body: Record<string, unknown>): HttpResponse => ({
    statusCode,
    body,
});

export default makeAuthzHttpHandler;
