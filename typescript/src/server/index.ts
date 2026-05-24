import {
  makeGraphAuthorizer,
  makeInMemoryTupleStore,
} from "../modules/authz/index.ts";
import { makeDemoTokenVerifier } from "../modules/authn/index.ts";
import { makeInMemoryDocumentRepository } from "../modules/db/index.ts";
import { demoTokens, seedRelationshipTuples } from "../modules/fixtures/index.ts";
import makeServerApp from "./compose.ts";

const readPort = (value: string | undefined, fallback: number): number => {
  if (value === undefined || value.trim() === "") return fallback;

  const portValue = Number(value);
  if (!Number.isInteger(portValue) || portValue < 1 || portValue > 65_535) {
    throw new Error(`Invalid PORT: ${value}`);
  }

  return portValue;
};

const tupleStore = makeInMemoryTupleStore({ seed: seedRelationshipTuples() });
const authorizer = makeGraphAuthorizer({ tupleStore });
const authenticator = makeDemoTokenVerifier({ tokens: demoTokens });
const repository = makeInMemoryDocumentRepository();
const port = readPort(process.env.PORT, 4000);

const app = await makeServerApp({ port, authenticator, authorizer, tupleStore, repository });

app.server.listen(app.port, () => {
  console.log(`TS ReBAC server listening on http://127.0.0.1:${app.port}`);
});
