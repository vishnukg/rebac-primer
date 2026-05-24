import type { ServerResponse } from "node:http";

export type JsonObject = Record<string, unknown>;

export const readJson = async (request: AsyncIterable<Buffer | string>): Promise<JsonObject> => {
    const chunks: Buffer[] = [];

    for await (const chunk of request) {
        chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
    }

    if (chunks.length === 0) return {};

    const parsed: unknown = JSON.parse(Buffer.concat(chunks).toString("utf8"));
    if (!isJsonObject(parsed)) {
        throw new Error("Request body must be a JSON object");
    }

    return parsed;
};

export const writeJson = (
    response: ServerResponse,
    statusCode: number,
    body: JsonObject,
): void => {
    response.writeHead(statusCode, { "content-type": "application/json" });
    response.end(JSON.stringify(body, null, 2));
};

export const isJsonObject = (value: unknown): value is JsonObject =>
    typeof value === "object" && value !== null && !Array.isArray(value);

export const stringField = (body: JsonObject, field: string): string => {
    const value = body[field];
    if (typeof value !== "string" || value.trim().length === 0) {
        throw new Error(`Missing string field: ${field}`);
    }
    return value;
};
