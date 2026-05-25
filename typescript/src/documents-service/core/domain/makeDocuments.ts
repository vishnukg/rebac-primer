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

const makeDocuments = ({ repository, authzClient }: DocumentsCfg): Documents => {
    const create = makeCreateDocument({ repository, authzClient });
    const read   = makeReadDocument({ repository, authzClient });
    const update = makeUpdateDocument({ repository, authzClient });
    return { create, read, update };
};

export default makeDocuments;
