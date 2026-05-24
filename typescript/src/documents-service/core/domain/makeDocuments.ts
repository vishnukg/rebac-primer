import type { AuthzClient } from "../ports/authzClient.ts";
import type { DocumentRepository } from "../ports/documentRepository.ts";
import type { Documents } from "./types.ts";
import makeCreateDocument from "./makeCreateDocument.ts";
import makeReadDocument from "./makeReadDocument.ts";
import makeUpdateDocument from "./makeUpdateDocument.ts";

type DocumentsCfg = {
    repository:  DocumentRepository;
    authzClient: AuthzClient;
};

const makeDocuments = ({ repository, authzClient }: DocumentsCfg): Documents => ({
    create: makeCreateDocument({ repository, authzClient }),
    read:   makeReadDocument({ repository, authzClient }),
    update: makeUpdateDocument({ repository, authzClient }),
});

export default makeDocuments;
