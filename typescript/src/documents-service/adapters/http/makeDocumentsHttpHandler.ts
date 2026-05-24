// HTTP adapter for the Documents service.
//
// Routes:
//   GET  /health
//   GET  /whoami      Authorization: Bearer <token>
//   POST /documents   Bearer token + body { id, title, body, workspaceId }
//   GET  /documents/:id  Bearer token
//   PATCH /documents/:id Bearer token + body { body }

import type {
    Authenticator,
    Documents,
} from "../../core/index.ts";
import {
    isAuthenticationError,
    isForbiddenError,
    isDocumentNotFoundError,
} from "../../core/index.ts";
import { workspace } from "../../../shared/rebac.ts";
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

export type DocumentsHttpHandler = (req: HttpRequest) => Promise<HttpResponse>;

type DocumentsHttpHandlerCfg = {
    authenticator: Authenticator;
    documents:     Documents;
};

const makeDocumentsHttpHandler = ({
    authenticator,
    documents,
}: DocumentsHttpHandlerCfg): DocumentsHttpHandler => {
    const handle: DocumentsHttpHandler = async request => {
        try {
            if (request.method === "GET" && request.path === "/health") {
                return json(200, { ok: true });
            }

            if (request.method === "GET" && request.path === "/whoami") {
                const authed = await authenticator.verifyAccessToken(request.authorization);
                return json(200, { user: authed.subject, scopes: authed.scopes });
            }

            if (request.method === "POST" && request.path === "/documents") {
                const authed = await authenticator.verifyAccessToken(request.authorization);
                const body   = requireBody(request.body);
                const doc    = await documents.create({
                    id:        stringField(body, "id"),
                    title:     stringField(body, "title"),
                    body:      stringField(body, "body"),
                    workspace: workspace(stringField(body, "workspaceId")),
                    actor:     authed.subject,
                });
                return json(201, { document: doc });
            }

            const docId = matchDocumentPath(request.path);

            if (docId && request.method === "GET") {
                const authed = await authenticator.verifyAccessToken(request.authorization);
                return json(200, {
                    document: await documents.read({ id: docId, actor: authed.subject }),
                });
            }

            if (docId && request.method === "PATCH") {
                const authed = await authenticator.verifyAccessToken(request.authorization);
                const body   = requireBody(request.body);
                return json(200, {
                    document: await documents.update({
                        id:    docId,
                        body:  stringField(body, "body"),
                        actor: authed.subject,
                    }),
                });
            }

            return json(404, { error: "Route not found" });
        } catch (error) {
            return toErrorResponse(error);
        }
    };

    return handle;
};

const requireBody = (body: unknown): Record<string, unknown> => {
    if (!isJsonObject(body)) throw new Error("Request body must be a JSON object");
    return body;
};

const matchDocumentPath = (pathname: string): string | undefined =>
    /^\/documents\/([^/]+)$/.exec(pathname)?.[1];

const toErrorResponse = (error: unknown): HttpResponse => {
    if (isAuthenticationError(error))   return json(401, { error: error.message });
    if (isForbiddenError(error))        return json(403, { error: error.message });
    if (isDocumentNotFoundError(error)) return json(404, { error: error.message });
    const message = error instanceof Error ? error.message : "Unknown error";
    return json(400, { error: message });
};

const json = (statusCode: number, body: Record<string, unknown>): HttpResponse => ({
    statusCode,
    body,
});

export default makeDocumentsHttpHandler;
