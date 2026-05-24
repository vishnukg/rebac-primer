import { createServer, type IncomingMessage, type Server } from "node:http";
import { readJson, writeJson } from "./json.ts";
import type { HttpHandler, HttpRequest } from "./makeHttpHandler.ts";

type HttpServerCfg = {
    handler: HttpHandler;
};

const makeHttpServer = ({ handler }: HttpServerCfg): Server =>
    createServer(async (request, response) => {
        const result = await handler(await toHttpRequest(request));
        writeJson(response, result.statusCode, result.body);
    });

const toHttpRequest = async (request: IncomingMessage): Promise<HttpRequest> => {
    const url = new URL(request.url ?? "/", "http://localhost");
    const base = {
        method:        request.method ?? "GET",
        path:          url.pathname,
        query:         url.searchParams,
        authorization: request.headers.authorization,
    };

    if (request.method === "POST" || request.method === "PATCH") {
        return { ...base, body: await readJson(request) };
    }

    return base;
};

export default makeHttpServer;
