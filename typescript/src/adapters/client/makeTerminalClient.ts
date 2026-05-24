// Terminal client — interactive CLI that demonstrates authn + authz end-to-end.
//
// Flow mirrors what a real client does:
//   1. Present credentials (enter a demo bearer token)
//   2. Verify identity via /whoami (authn)
//   3. Perform document operations (authz checked per request on the server)
//
// Demo tokens: demo-token-alice (editor), demo-token-bob (viewer), demo-token-casey (denied)

import type { DocumentsClient } from "./makeHttpDocumentsClient.ts";

export type QuestionTerminal = {
    question: (prompt: string) => Promise<string>;
};

export type TerminalClient = {
    run: () => Promise<void>;
};

type TerminalClientCfg = {
    client:   DocumentsClient;
    terminal: QuestionTerminal;
    write:    (message: string) => void;
};

const makeTerminalClient = ({ client, terminal, write }: TerminalClientCfg): TerminalClient => {
    const run = async (): Promise<void> => {
        write("TS ReBAC client\n");

        if (!(await client.health())) {
            throw new Error("Server is not reachable. Is it running?");
        }

        // ── Step 1: authn — identify yourself with a bearer token ─────────────
        write("Demo tokens:");
        write("  demo-token-alice  (platform team member → can create, read, edit)");
        write("  demo-token-bob    (workspace viewer → can read, cannot edit)");
        write("  demo-token-casey  (no relationships → denied)\n");

        const token = (await terminal.question("Enter token: ")).trim();

        // ── Step 2: verify identity via /whoami ───────────────────────────────
        try {
            const subject = await client.whoami(token);
            write(`Authenticated as: ${subject}`);
            write("");
        } catch {
            write("Authentication failed: invalid token");
            return;
        }

        // ── Step 3: document operations (authz enforced server-side) ──────────
        let running = true;
        while (running) {
            write("1. Create document");
            write("2. Read document");
            write("3. Update document");
            write("4. Exit");

            const choice = (await terminal.question("\nChoose: ")).trim();
            write("");

            if (choice === "1")      await createDocument(token);
            else if (choice === "2") await readDocument(token);
            else if (choice === "3") await updateDocument(token);
            else if (choice === "4") running = false;
            else                     write("Unknown choice");
        }
    };

    const createDocument = async (token: string): Promise<void> => {
        try {
            const id          = (await terminal.question("Document id: ")).trim();
            const title       = (await terminal.question("Title: ")).trim();
            const body        = (await terminal.question("Body: ")).trim();
            const workspaceId = (await terminal.question("Workspace id (e.g. productWorkspace): ")).trim();

            const doc = await client.createDocument({ id, title, body, workspaceId }, token);
            write(`\nCreated: ${doc.title} (${doc.id})`);
            write(`Workspace: ${doc.workspace}`);
            write(`Updated by: ${doc.updatedBy}\n`);
        } catch (error) {
            write(`\nError: ${errorMessage(error)}\n`);
        }
    };

    const readDocument = async (token: string): Promise<void> => {
        try {
            const id  = (await terminal.question("Document id: ")).trim();
            const doc = await client.readDocument(id, token);
            write(`\n${doc.title}`);
            write(doc.body);
            write(`Updated by: ${doc.updatedBy}\n`);
        } catch (error) {
            write(`\nError: ${errorMessage(error)}\n`);
        }
    };

    const updateDocument = async (token: string): Promise<void> => {
        try {
            const id      = (await terminal.question("Document id: ")).trim();
            const newBody = (await terminal.question("New body: ")).trim();
            const doc     = await client.updateDocument(id, newBody, token);
            write(`\nUpdated: ${doc.id}`);
            write(`Updated by: ${doc.updatedBy}\n`);
        } catch (error) {
            write(`\nError: ${errorMessage(error)}\n`);
        }
    };

    return { run };
};

const errorMessage = (error: unknown): string =>
    error instanceof Error ? error.message : "unknown error";

export default makeTerminalClient;
