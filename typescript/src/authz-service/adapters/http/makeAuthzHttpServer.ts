import { createServer, type Server, type IncomingMessage, type ServerResponse } from "node:http";
import type { AuthzHttpHandler } from "./makeAuthzHttpHandler.ts";

type AuthzHttpServerCfg = {
    handler: AuthzHttpHandler;
};

const makeAuthzHttpServer = ({ handler }: AuthzHttpServerCfg): Server => {
    const server = createServer(async (req: IncomingMessage, res: ServerResponse) => {
        const url    = new URL(req.url ?? "/", `http://localhost`);
        const body   = await readBody(req);
        const method = req.method ?? "GET";

        const response = await handler({
            method,
            path:  url.pathname,
            query: url.searchParams,
            body:  body ? safeParseJson(body) : undefined,
        });

        res.writeHead(response.statusCode, { "content-type": "application/json" });
        res.end(JSON.stringify(response.body));
    });

    return server;
};

const readBody = (req: IncomingMessage): Promise<string> => {
    // Promise.withResolvers (ES2024) hands back the resolve/reject functions so
    // they can be wired straight to the stream's events — no nested executor, and
    // a natural place to reject on a stream error.
    const { promise, resolve, reject } = Promise.withResolvers<string>();
    const chunks: Buffer[] = [];
    req.on("data", c => chunks.push(c));
    req.on("end", () => resolve(Buffer.concat(chunks).toString()));
    req.on("error", reject);
    return promise;
};

const safeParseJson = (text: string): unknown => {
    try { return JSON.parse(text); }
    catch { return undefined; }
};

export default makeAuthzHttpServer;
