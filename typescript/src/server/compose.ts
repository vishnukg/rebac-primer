import {
  makeCreateDocument,
  makeDocuments,
  makeReadDocument,
  makeRequireAllowed,
  makeUpdateDocument,
} from "../modules/documents/index.ts";
import type { Authenticator } from "../modules/authn/index.ts";
import type { Authorizer, TupleStore } from "../modules/authz/index.ts";
import type { DocumentRepository } from "../modules/documents/index.ts";
import { makeHttpHandler, makeHttpServer } from "../modules/http/index.ts";
import { seedRoadmapDocument } from "../modules/fixtures/index.ts";

type ServerAppCfg = {
  port:          number;
  authenticator: Authenticator;
  authorizer:    Authorizer;
  tupleStore:    TupleStore;
  repository:    DocumentRepository;
};

const makeServerApp = async ({ port, authenticator, authorizer, repository }: ServerAppCfg) => {
  const requireAllowed = makeRequireAllowed({ authorizer });
  const create = makeCreateDocument({ repository, requireAllowed });
  const read = makeReadDocument({ repository, requireAllowed });
  const update = makeUpdateDocument({ repository, requireAllowed });
  const documents = makeDocuments({ create, read, update });
  const handler = makeHttpHandler({ authenticator, documents });
  const server = makeHttpServer({ handler });

  await documents.create(seedRoadmapDocument);

  return { port, server, documents, authorizer };
};

export default makeServerApp;
