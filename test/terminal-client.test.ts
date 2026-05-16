import { describe, expect, it } from "vitest";
import type { DocumentsClient } from "../src/client/api-client.js";
import { TerminalClient, type QuestionTerminal } from "../src/client/terminal-client.js";
import type { CollaborativeDocument } from "../src/domain/document.js";

describe("TerminalClient", () => {
  it("given_read_choice_when_terminal_client_runs_then_document_is_read_and_printed", async () => {
    // Arrange
    const prompts: string[] = [];
    const writes: string[] = [];
    const answers = ["1", "workspaceViewer", "3"];
    const terminal: QuestionTerminal = {
      question: async (prompt) => {
        prompts.push(prompt);
        const answer = answers.shift();
        if (answer === undefined) {
          throw new Error("No answer arranged for prompt");
        }
        return answer;
      }
    };
    const document: CollaborativeDocument = {
      id: "roadmapDocument",
      title: "Roadmap",
      body: "Read the tutorial",
      workspace: "workspace:productWorkspace",
      updatedBy: "user:workspaceEditor"
    };
    const client: DocumentsClient = {
      health: async () => true,
      readDocument: async (id, actorId) => {
        expect(id).toBe("roadmapDocument");
        expect(actorId).toBe("workspaceViewer");
        return document;
      },
      updateDocument: async () => {
        throw new Error("Update was not expected");
      }
    };
    const terminalClient = new TerminalClient({
      client,
      terminal,
      write: (message) => writes.push(message)
    });

    // Act
    await terminalClient.run();

    // Assert
    expect(prompts).toEqual(["Choose: ", "Actor id: ", "Choose: "]);
    expect(writes).toContain("\nRoadmap");
    expect(writes).toContain("Read the tutorial");
    expect(writes).toContain("updated by user:workspaceEditor");
  });

  it("given_update_choice_when_terminal_client_runs_then_document_is_updated_and_printed", async () => {
    // Arrange
    const prompts: string[] = [];
    const writes: string[] = [];
    const answers = ["2", "workspaceEditor", "Ship the primer", "3"];
    const terminal: QuestionTerminal = {
      question: async (prompt) => {
        prompts.push(prompt);
        const answer = answers.shift();
        if (answer === undefined) {
          throw new Error("No answer arranged for prompt");
        }
        return answer;
      }
    };
    const document: CollaborativeDocument = {
      id: "roadmapDocument",
      title: "Roadmap",
      body: "Ship the primer",
      workspace: "workspace:productWorkspace",
      updatedBy: "user:workspaceEditor"
    };
    const client: DocumentsClient = {
      health: async () => true,
      readDocument: async () => {
        throw new Error("Read was not expected");
      },
      updateDocument: async (id, actorId, body) => {
        expect(id).toBe("roadmapDocument");
        expect(actorId).toBe("workspaceEditor");
        expect(body).toBe("Ship the primer");
        return document;
      }
    };
    const terminalClient = new TerminalClient({
      client,
      terminal,
      write: (message) => writes.push(message)
    });

    // Act
    await terminalClient.run();

    // Assert
    expect(prompts).toEqual(["Choose: ", "Actor id: ", "New body: ", "Choose: "]);
    expect(writes).toContain("Updated roadmapDocument; updated by user:workspaceEditor");
  });

  it("given_denied_read_when_terminal_client_runs_then_denial_message_is_written_and_loop_continues", async () => {
    // Arrange
    const writes: string[] = [];
    const answers = ["1", "outsideCollaborator", "3"];
    const terminal: QuestionTerminal = {
      question: async () => {
        const answer = answers.shift();
        if (answer === undefined) throw new Error("No answer arranged");
        return answer;
      }
    };
    const client: DocumentsClient = {
      health: async () => true,
      readDocument: async () => {
        throw new Error("user:outsideCollaborator cannot read document:roadmapDocument");
      },
      updateDocument: async () => {
        throw new Error("Update was not expected");
      }
    };
    const terminalClient = new TerminalClient({
      client,
      terminal,
      write: (message) => writes.push(message)
    });

    // Act
    await terminalClient.run();

    // Assert
    expect(writes).toContain("Denied: user:outsideCollaborator cannot read document:roadmapDocument");
  });

  it("given_unhealthy_server_when_terminal_client_runs_then_health_check_error_is_thrown", async () => {
    // Arrange
    const terminal: QuestionTerminal = {
      question: async () => {
        throw new Error("Prompt was not expected");
      }
    };
    const client: DocumentsClient = {
      health: async () => false,
      readDocument: async () => {
        throw new Error("Read was not expected");
      },
      updateDocument: async () => {
        throw new Error("Update was not expected");
      }
    };
    const terminalClient = new TerminalClient({
      client,
      terminal,
      write: () => undefined
    });

    // Act
    const runPromise = terminalClient.run();

    // Assert
    await expect(runPromise).rejects.toThrow("Server health check failed");
  });
});
