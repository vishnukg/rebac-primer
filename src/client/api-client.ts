import type { CollaborativeDocument } from "../domain/document.js";
import { isJsonObject } from "../http/json.js";

export interface DocumentsClient {
  health(): Promise<boolean>;
  readDocument(id: string, actorId: string): Promise<CollaborativeDocument>;
  updateDocument(id: string, actorId: string, body: string): Promise<CollaborativeDocument>;
}

export type Fetcher = (input: URL, init?: RequestInit) => Promise<Response>;

export class RebacApiClient implements DocumentsClient {
  constructor(
    private readonly baseUrl: string,
    private readonly fetcher: Fetcher = fetch
  ) {}

  async health(): Promise<boolean> {
    const response = await this.fetcher(new URL("/health", this.baseUrl));
    return response.ok;
  }

  async readDocument(id: string, actorId: string): Promise<CollaborativeDocument> {
    const url = new URL(`/documents/${id}`, this.baseUrl);
    url.searchParams.set("actorId", actorId);

    const body = await this.request(url, {
      method: "GET"
    });
    return documentFromResponse(body);
  }

  async updateDocument(id: string, actorId: string, body: string): Promise<CollaborativeDocument> {
    const result = await this.request(
      new URL(`/documents/${id}`, this.baseUrl),
      {
        method: "PATCH",
        body: JSON.stringify({ actorId, body })
      }
    );
    return documentFromResponse(result);
  }

  private async request(url: URL, init: RequestInit): Promise<unknown> {
    const response = await this.fetcher(url, {
      ...init,
      headers: {
        "content-type": "application/json",
        ...init.headers
      }
    });
    const body: unknown = await response.json();

    if (!response.ok) {
      const message = hasErrorMessage(body) ? body.error : response.statusText;
      throw new Error(message);
    }

    return body;
  }
}

function documentFromResponse(value: unknown): CollaborativeDocument {
  if (!isJsonObject(value) || !isCollaborativeDocument(value.document)) {
    throw new Error("Response body did not contain a document");
  }

  return value.document;
}

function isCollaborativeDocument(value: unknown): value is CollaborativeDocument {
  return (
    isJsonObject(value) &&
    typeof value.id === "string" &&
    typeof value.title === "string" &&
    typeof value.body === "string" &&
    typeof value.workspace === "string" &&
    value.workspace.startsWith("workspace:") &&
    typeof value.updatedBy === "string" &&
    value.updatedBy.startsWith("user:")
  );
}

function hasErrorMessage(value: unknown): value is { error: string } {
  return (
    typeof value === "object" &&
    value !== null &&
    "error" in value &&
    typeof value.error === "string"
  );
}
