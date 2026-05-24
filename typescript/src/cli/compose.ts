import { stdin as input, stdout as output } from "node:process";
import { createInterface } from "node:readline/promises";
import makeHttpDocumentsClient from "../documents-service/adapters/client/makeHttpDocumentsClient.ts";
import makeTerminalClient from "../documents-service/adapters/client/makeTerminalClient.ts";

type CliAppCfg = {
    env?: NodeJS.ProcessEnv;
};

const makeCliApp = ({ env = process.env }: CliAppCfg = {}) => {
    const terminal = createInterface({ input, output });
    const client   = makeHttpDocumentsClient({
        baseUrl: env.REBAC_API_URL ?? "http://127.0.0.1:4000",
    });
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
