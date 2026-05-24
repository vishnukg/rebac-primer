import type { CollaborativeDocument } from "../documents/index.ts";
import { isJsonObject } from "../http/index.ts";
import type { DocumentsClient, Fetcher } from "./types.ts";

type HttpDocumentsClientCfg = {
  baseUrl: string;
  fetcher?: Fetcher;
};

const makeHttpDocumentsClient = ({ baseUrl, fetcher = fetch }: HttpDocumentsClientCfg): DocumentsClient => {
  const request = async (url: URL, init: RequestInit): Promise<unknown> => {
    const response = await fetcher(url, {
      ...init,
      headers: {
        "content-type": "application/json",
        ...init.headers,
      },
    });
    const body: unknown = await response.json();

    if (!response.ok) {
      throw new Error(hasErrorMessage(body) ? body.error : response.statusText);
    }

    return body;
  };

  const health = async (): Promise<boolean> => {
    const response = await fetcher(new URL("/health", baseUrl));
    return response.ok;
  };

  const whoami = async (token: string): Promise<string> => {
    const body = await request(new URL("/whoami", baseUrl), {
      method:  "GET",
      headers: { authorization: `Bearer ${token}` },
    });

    if (!isJsonObject(body) || typeof body.user !== "string") {
      throw new Error("Response body did not contain a user");
    }

    return body.user;
  };

  const readDocument = async (id: string, actorId: string): Promise<CollaborativeDocument> => {
    const url = new URL(`/documents/${id}`, baseUrl);
    url.searchParams.set("actorId", actorId);
    return documentFromResponse(await request(url, { method: "GET" }));
  };

  const updateDocument = async (id: string, actorId: string, body: string): Promise<CollaborativeDocument> =>
    documentFromResponse(await request(new URL(`/documents/${id}`, baseUrl), {
      method: "PATCH",
      body:   JSON.stringify({ actorId, body }),
    }));

  return { health, whoami, readDocument, updateDocument };
};

const documentFromResponse = (value: unknown): CollaborativeDocument => {
  if (!isJsonObject(value) || !isCollaborativeDocument(value.document)) {
    throw new Error("Response body did not contain a document");
  }
  return value.document;
};

const isCollaborativeDocument = (value: unknown): value is CollaborativeDocument =>
  isJsonObject(value) &&
  typeof value.id === "string" &&
  typeof value.title === "string" &&
  typeof value.body === "string" &&
  typeof value.workspace === "string" &&
  value.workspace.startsWith("workspace:") &&
  typeof value.updatedBy === "string" &&
  value.updatedBy.startsWith("user:");

const hasErrorMessage = (value: unknown): value is { error: string } =>
  isJsonObject(value) && typeof value.error === "string";

export default makeHttpDocumentsClient;
