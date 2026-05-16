import type { DocumentsClient } from "./api-client.js";

export interface QuestionTerminal {
  question(prompt: string): Promise<string>;
}

export type TerminalClientConfig = Readonly<{
  client: DocumentsClient;
  terminal: QuestionTerminal;
  write: (message: string) => void;
}>;

export class TerminalClient {
  constructor(private readonly config: TerminalClientConfig) {}

  async run(): Promise<void> {
    this.config.write("TS ReBAC client");
    this.config.write("Enter an actor id when prompted. Available actors:");
    this.config.write("  workspaceEditor      — can read and edit (workspace editor via team membership)");
    this.config.write("  workspaceViewer      — can read only");
    this.config.write("  outsideCollaborator  — denied (no path through the graph)");

    const healthy = await this.config.client.health();
    if (!healthy) {
      throw new Error("Server health check failed");
    }

    let running = true;
    while (running) {
      this.config.write("\n1. Read roadmap document");
      this.config.write("2. Update roadmap document");
      this.config.write("3. Exit");

      const choice = await this.config.terminal.question("Choose: ");

      if (choice === "1") {
        await this.readRoadmapDocument();
      } else if (choice === "2") {
        await this.updateRoadmapDocument();
      } else if (choice === "3") {
        running = false;
      } else {
        this.config.write("Unknown choice");
      }
    }
  }

  private async readRoadmapDocument(): Promise<void> {
    try {
      const actorId = await this.config.terminal.question("Actor id: ");
      const document = await this.config.client.readDocument("roadmapDocument", actorId);
      this.config.write(`\n${document.title}`);
      this.config.write(document.body);
      this.config.write(`updated by ${document.updatedBy}`);
    } catch (error) {
      this.config.write(`Denied: ${error instanceof Error ? error.message : "unknown error"}`);
    }
  }

  private async updateRoadmapDocument(): Promise<void> {
    try {
      const actorId = await this.config.terminal.question("Actor id: ");
      const body = await this.config.terminal.question("New body: ");
      const document = await this.config.client.updateDocument("roadmapDocument", actorId, body);
      this.config.write(`Updated ${document.id}; updated by ${document.updatedBy}`);
    } catch (error) {
      this.config.write(`Denied: ${error instanceof Error ? error.message : "unknown error"}`);
    }
  }
}
