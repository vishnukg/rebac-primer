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
        write("TS ReBAC client");
        write("Try actors: alice can edit, bob can read, casey is denied.");

        if (!(await client.health())) {
            throw new Error("Server health check failed");
        }

        let running = true;
        while (running) {
            write("\n1. Read roadmap document");
            write("2. Update roadmap document");
            write("3. Exit");

            const choice = await terminal.question("Choose: ");
            if (choice === "1")      await readRoadmapDocument();
            else if (choice === "2") await updateRoadmapDocument();
            else if (choice === "3") running = false;
            else                     write("Unknown choice");
        }
    };

    const readRoadmapDocument = async (): Promise<void> => {
        try {
            const actorId = await terminal.question("Actor id: ");
            const doc = await client.readDocument("roadmapDocument", actorId);
            write(`\n${doc.title}`);
            write(doc.body);
            write(`updated by ${doc.updatedBy}`);
        } catch (error) {
            write(`Denied: ${error instanceof Error ? error.message : "unknown error"}`);
        }
    };

    const updateRoadmapDocument = async (): Promise<void> => {
        try {
            const actorId = await terminal.question("Actor id: ");
            const newBody = await terminal.question("New body: ");
            const doc = await client.updateDocument("roadmapDocument", actorId, newBody);
            write(`Updated ${doc.id}; updated by ${doc.updatedBy}`);
        } catch (error) {
            write(`Denied: ${error instanceof Error ? error.message : "unknown error"}`);
        }
    };

    return { run };
};

export default makeTerminalClient;
