import type { ServerResponse } from "node:http";

export type JsonObject = Record<string, unknown>;

export async function readJson(request: AsyncIterable<Buffer | string>): Promise<JsonObject> {
  const chunks: Buffer[] = [];

  for await (const chunk of request) {
    chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
  }

  if (chunks.length === 0) {
    return {};
  }

  const rawBody = Buffer.concat(chunks).toString("utf8");
  const parsed: unknown = JSON.parse(rawBody);

  if (!isJsonObject(parsed)) {
    throw new Error("Request body must be a JSON object");
  }

  return parsed;
}

export function writeJson(
  response: ServerResponse,
  statusCode: number,
  body: JsonObject
): void {
  response.writeHead(statusCode, { "content-type": "application/json" });
  response.end(JSON.stringify(body, null, 2));
}

export function isJsonObject(value: unknown): value is JsonObject {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function stringField(body: JsonObject, field: string): string {
  const value = body[field];
  if (typeof value !== "string" || value.trim().length === 0) {
    throw new Error(`Missing string field: ${field}`);
  }

  return value;
}
