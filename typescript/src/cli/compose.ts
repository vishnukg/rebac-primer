import { stdin as input, stdout as output } from "node:process";
import { createInterface } from "node:readline/promises";
import { makeHttpDocumentsClient, makeTerminalClient } from "../modules/client/index.ts";

const makeCliApp = (env: NodeJS.ProcessEnv = process.env) => {
  const terminal = createInterface({ input, output });
  const client = makeHttpDocumentsClient({ baseUrl: env.REBAC_API_URL ?? "http://127.0.0.1:4000" });
  const terminalClient = makeTerminalClient({
    client,
    terminal,
    write: message => console.log(message),
  });

  return {
    terminal,
    run: terminalClient.run,
  };
};

export default makeCliApp;
