import { createServer, type IncomingMessage, type Server } from "node:http";
import type { DocumentWorkflow } from "../domain/service.js";
import { handleHttpRequest } from "./handler.js";
import { readJson, writeJson } from "./json.js";

export type HttpServerConfig = Readonly<{
  documents: DocumentWorkflow;
}>;

export function createHttpServer(config: HttpServerConfig): Server {
  return createServer(async (request, response) => {
    const result = await handleHttpRequest(config, await toHttpRequest(request));
    writeJson(response, result.statusCode, result.body);
  });
}

async function toHttpRequest(request: IncomingMessage): Promise<{
  method: string;
  path: string;
  query: URLSearchParams;
  body?: unknown;
}> {
  const url = new URL(request.url ?? "/", "http://localhost");
  const base = {
    method: request.method ?? "GET",
    path: url.pathname,
    query: url.searchParams
  };

  if (request.method === "POST" || request.method === "PATCH") {
    return { ...base, body: await readJson(request) };
  }

  return base;
}
