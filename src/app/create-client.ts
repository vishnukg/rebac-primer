import { stdin as input, stdout as output } from "node:process";
import { createInterface } from "node:readline/promises";
import { RebacApiClient } from "../client/api-client.js";
import { TerminalClient } from "../client/terminal-client.js";

export type ClientApp = Readonly<{
  terminal: ReturnType<typeof createInterface>;
  run: () => Promise<void>;
}>;

export function createClientApp(env: NodeJS.ProcessEnv = process.env): ClientApp {
  const terminal = createInterface({ input, output });
  const client = new RebacApiClient(env.REBAC_API_URL ?? "http://127.0.0.1:4000");
  const terminalClient = new TerminalClient({
    client,
    terminal,
    write: (message) => console.log(message)
  });

  return {
    terminal,
    run: () => terminalClient.run()
  };
}
