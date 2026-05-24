import type { CollaborativeDocument } from "../../core/index.ts";
import { isJsonObject } from "../http/json.ts";

export type Fetcher = (input: URL, init?: RequestInit) => Promise<Response>;

export type DocumentsClient = {
    health:         () => Promise<boolean>;
    whoami:         (token: string) => Promise<string>;
    createDocument: (input: CreateDocumentClientInput, token: string) => Promise<CollaborativeDocument>;
    readDocument:   (id: string, token: string) => Promise<CollaborativeDocument>;
    updateDocument: (id: string, body: string, token: string) => Promise<CollaborativeDocument>;
};

export type CreateDocumentClientInput = {
    id:          string;
    title:       string;
    body:        string;
    workspaceId: string;
};

type HttpDocumentsClientCfg = {
    baseUrl:  string;
    fetcher?: Fetcher;
};

const makeHttpDocumentsClient = ({
    baseUrl,
    fetcher = fetch,
}: HttpDocumentsClientCfg): DocumentsClient => {
    const request = async (url: URL, init: RequestInit): Promise<unknown> => {
        const response = await fetcher(url, {
            ...init,
            headers: { "content-type": "application/json", ...init.headers },
        });
        const body: unknown = await response.json();
        if (!response.ok) {
            throw new Error(hasErrorMessage(body) ? body.error : response.statusText);
        }
        return body;
    };

    const bearerHeader = (token: string): { authorization: string } => ({
        authorization: `Bearer ${token}`,
    });

    const health = async (): Promise<boolean> => {
        const response = await fetcher(new URL("/health", baseUrl));
        return response.ok;
    };

    const whoami = async (token: string): Promise<string> => {
        const body = await request(new URL("/whoami", baseUrl), {
            method:  "GET",
            headers: bearerHeader(token),
        });
        if (!isJsonObject(body) || typeof body.user !== "string") {
            throw new Error("Response body did not contain a user");
        }
        return body.user;
    };

    const createDocument = async (
        input: CreateDocumentClientInput,
        token: string,
    ): Promise<CollaborativeDocument> =>
        documentFromResponse(
            await request(new URL("/documents", baseUrl), {
                method:  "POST",
                headers: bearerHeader(token),
                body:    JSON.stringify(input),
            }),
        );

    const readDocument = async (id: string, token: string): Promise<CollaborativeDocument> =>
        documentFromResponse(
            await request(new URL(`/documents/${id}`, baseUrl), {
                method:  "GET",
                headers: bearerHeader(token),
            }),
        );

    const updateDocument = async (
        id: string,
        body: string,
        token: string,
    ): Promise<CollaborativeDocument> =>
        documentFromResponse(
            await request(new URL(`/documents/${id}`, baseUrl), {
                method:  "PATCH",
                headers: bearerHeader(token),
                body:    JSON.stringify({ body }),
            }),
        );

    return { health, whoami, createDocument, readDocument, updateDocument };
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
