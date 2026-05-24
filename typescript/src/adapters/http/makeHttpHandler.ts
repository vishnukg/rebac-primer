import {
    AuthenticationError,
    DocumentNotFoundError,
    ForbiddenError,
    user,
    workspace,
} from "../../core/index.ts";
import type { Authenticator, Documents } from "../../core/index.ts";
import { isJsonObject, stringField } from "./json.ts";

export type HttpRequest = {
    method:        string;
    path:          string;
    query:         URLSearchParams;
    authorization: string | undefined;
    body?:         unknown;
};

export type HttpResponse = {
    statusCode: number;
    body:       Record<string, unknown>;
};

export type HttpHandler = (request: HttpRequest) => Promise<HttpResponse>;

type HttpHandlerCfg = {
    authenticator: Authenticator;
    documents:     Documents;
};

const makeHttpHandler = ({ authenticator, documents }: HttpHandlerCfg): HttpHandler => {
    const handle: HttpHandler = async request => {
        try {
            if (request.method === "GET" && request.path === "/health") {
                return json(200, { ok: true });
            }

            if (request.method === "GET" && request.path === "/whoami") {
                const authed = await authenticator.verifyAccessToken(request.authorization);
                return json(200, { user: authed.subject, scopes: authed.scopes });
            }

            if (request.method === "POST" && request.path === "/documents") {
                const body  = requireBody(request.body);
                const authed = await authenticator.verifyAccessToken(request.authorization);
                const created = await documents.create({
                    id:        stringField(body, "id"),
                    title:     stringField(body, "title"),
                    body:      stringField(body, "body"),
                    workspace: workspace(stringField(body, "workspaceId")),
                    actor:     authed.subject,
                });
                return json(201, { document: created });
            }

            const documentId = matchDocumentPath(request.path);

            if (documentId && request.method === "GET") {
                const actor = await resolveActor(
                    request.authorization,
                    request.query.get("actorId") ?? undefined,
                );
                return json(200, { document: await documents.read(documentId, actor) });
            }

            if (documentId && request.method === "PATCH") {
                const body  = requireBody(request.body);
                const actor = await resolveActor(
                    request.authorization,
                    optionalString(body, "actorId"),
                );
                const updated = await documents.update({
                    id:    documentId,
                    body:  stringField(body, "body"),
                    actor,
                });
                return json(200, { document: updated });
            }

            return json(404, { error: "Route not found" });
        } catch (error) {
            return toErrorResponse(error);
        }
    };

    // If actorIdOverride is provided (demo/testing convenience), use it directly.
    // Otherwise verify the bearer token and use the authenticated subject.
    const resolveActor = async (
        authorization: string | undefined,
        actorIdOverride: string | undefined,
    ) => {
        if (actorIdOverride) return user(actorIdOverride);
        return (await authenticator.verifyAccessToken(authorization)).subject;
    };

    return handle;
};

const requireBody = (body: unknown): Record<string, unknown> => {
    if (!isJsonObject(body)) throw new Error("Request body must be a JSON object");
    return body;
};

const optionalString = (body: Record<string, unknown>, field: string): string | undefined => {
    const value = body[field];
    if (value === undefined) return undefined;
    if (typeof value !== "string" || value.trim().length === 0) {
        throw new Error(`Invalid string field: ${field}`);
    }
    return value;
};

const matchDocumentPath = (pathname: string): string | undefined =>
    /^\/documents\/([^/]+)$/.exec(pathname)?.[1];

const toErrorResponse = (error: unknown): HttpResponse => {
    if (error instanceof AuthenticationError)  return json(401, { error: error.message });
    if (error instanceof ForbiddenError)       return json(403, { error: error.message });
    if (error instanceof DocumentNotFoundError) return json(404, { error: error.message });
    const message = error instanceof Error ? error.message : "Unknown error";
    return json(400, { error: message });
};

const json = (statusCode: number, body: Record<string, unknown>): HttpResponse => ({
    statusCode,
    body,
});

export default makeHttpHandler;
