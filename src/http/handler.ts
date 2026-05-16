import { user, workspace } from "../authz/types.js";
import { DocumentNotFoundError, ForbiddenError } from "../domain/document.js";
import type { DocumentWorkflow } from "../domain/service.js";
import { isJsonObject, stringField, type JsonObject } from "./json.js";

export type HttpRequest = Readonly<{
  method: string;
  path: string;
  query: URLSearchParams;
  body?: unknown;
}>;

export type HttpResponse = Readonly<{
  statusCode: number;
  body: JsonObject;
}>;

export type HttpHandlerConfig = Readonly<{
  documents: DocumentWorkflow;
}>;

export async function handleHttpRequest(
  config: HttpHandlerConfig,
  request: HttpRequest
): Promise<HttpResponse> {
  try {
    return await routeRequest(config.documents, request);
  } catch (error) {
    return errorResponse(error);
  }
}

async function routeRequest(
  documents: DocumentWorkflow,
  request: HttpRequest
): Promise<HttpResponse> {
  if (request.method === "GET" && request.path === "/health") {
    return json(200, { ok: true });
  }

  if (request.method === "POST" && request.path === "/documents") {
    const body = requiredBody(request.body);
    const created = await documents.create({
      id: stringField(body, "id"),
      title: stringField(body, "title"),
      body: stringField(body, "body"),
      workspace: workspace(stringField(body, "workspaceId")),
      actor: user(stringField(body, "actorId"))
    });
    return json(201, { document: created });
  }

  const documentId = matchDocumentPath(request.path);
  if (documentId && request.method === "GET") {
    const actorId = requiredQuery(request.query, "actorId");
    const found = await documents.read(documentId, user(actorId));
    return json(200, { document: found });
  }

  if (documentId && request.method === "PATCH") {
    const body = requiredBody(request.body);
    const updated = await documents.update({
      id: documentId,
      body: stringField(body, "body"),
      actor: user(stringField(body, "actorId"))
    });
    return json(200, { document: updated });
  }

  return json(404, { error: "Route not found" });
}

function requiredBody(body: unknown): JsonObject {
  if (!isJsonObject(body)) {
    throw new Error("Request body must be a JSON object");
  }

  return body;
}

function matchDocumentPath(pathname: string): string | undefined {
  const match = /^\/documents\/([^/]+)$/.exec(pathname);
  return match?.[1];
}

function requiredQuery(query: URLSearchParams, key: string): string {
  const value = query.get(key);
  if (!value) {
    throw new Error(`Missing query parameter: ${key}`);
  }

  return value;
}

function errorResponse(error: unknown): HttpResponse {
  if (error instanceof ForbiddenError) {
    return json(403, { error: error.message });
  }

  if (error instanceof DocumentNotFoundError) {
    return json(404, { error: error.message });
  }

  const message = error instanceof Error ? error.message : "Unknown error";
  return json(400, { error: message });
}

function json(statusCode: number, body: JsonObject): HttpResponse {
  return { statusCode, body };
}
