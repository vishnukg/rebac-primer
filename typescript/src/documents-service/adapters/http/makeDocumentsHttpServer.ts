import { createServer, type IncomingMessage, type ServerResponse } from "node:http";
import type { DocumentsHttpHandler } from "./makeDocumentsHttpHandler.ts";

type DocumentsHttpServerCfg = {
    handler: DocumentsHttpHandler;
};

const makeDocumentsHttpServer = ({ handler }: DocumentsHttpServerCfg) => {
    const server = createServer(async (req: IncomingMessage, res: ServerResponse) => {
        const url    = new URL(req.url ?? "/", `http://localhost`);
        const body   = await readBody(req);
        const method = req.method ?? "GET";

        const response = await handler({
            method,
            path:          url.pathname,
            query:         url.searchParams,
            authorization: req.headers.authorization,
            body:          body ? safeParseJson(body) : undefined,
        });

        res.writeHead(response.statusCode, { "content-type": "application/json" });
        res.end(JSON.stringify(response.body));
    });

    return server;
};

const readBody = (req: IncomingMessage): Promise<string> =>
    new Promise(resolve => {
        const chunks: Buffer[] = [];
        req.on("data", c => chunks.push(c));
        req.on("end", () => resolve(Buffer.concat(chunks).toString()));
    });

const safeParseJson = (text: string): unknown => {
    try { return JSON.parse(text); }
    catch { return undefined; }
};

export default makeDocumentsHttpServer;
