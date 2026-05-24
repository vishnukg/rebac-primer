import type { CollaborativeDocument } from "../documents/index.ts";

export type DocumentsClient = {
  health:         () => Promise<boolean>;
  whoami:         (token: string) => Promise<string>;
  readDocument:   (id: string, actorId: string) => Promise<CollaborativeDocument>;
  updateDocument: (id: string, actorId: string, body: string) => Promise<CollaborativeDocument>;
};

export type Fetcher = (input: URL, init?: RequestInit) => Promise<Response>;

export type QuestionTerminal = {
  question: (prompt: string) => Promise<string>;
};

export type TerminalClient = {
  run: () => Promise<void>;
};
