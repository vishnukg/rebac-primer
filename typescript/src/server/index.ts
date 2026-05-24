import makeGraphAuthorizer from "../adapters/authz/makeGraphAuthorizer.ts";
import makeInMemoryTupleStore from "../adapters/authz/makeInMemoryTupleStore.ts";
import makeDemoTokenVerifier from "../adapters/authn/makeDemoTokenVerifier.ts";
import makeInMemoryDocumentRepository from "../adapters/db/makeInMemoryDocumentRepository.ts";
import {
    demoTokens,
    seedRelationshipTuples,
    seedRoadmapDocument,
} from "../demo/fixtures.ts";
import makeServerApp from "./compose.ts";

const readPort = (value: string | undefined, fallback: number): number => {
    if (value === undefined || value.trim() === "") return fallback;
    const portValue = Number(value);
    if (!Number.isInteger(portValue) || portValue < 1 || portValue > 65_535) {
        throw new Error(`Invalid PORT: ${value}`);
    }
    return portValue;
};

const tupleStore    = makeInMemoryTupleStore({ seed: seedRelationshipTuples() });
const authorizer    = makeGraphAuthorizer({ tupleStore });
const authenticator = makeDemoTokenVerifier({ tokens: demoTokens });
const repository    = makeInMemoryDocumentRepository();
const port          = readPort(process.env.PORT, 4000);

const app = makeServerApp({ port, authenticator, authorizer, repository });

await app.documents.create(seedRoadmapDocument);

app.server.listen(port, () => {
    console.log(`TS ReBAC server listening on http://127.0.0.1:${port}`);
});
