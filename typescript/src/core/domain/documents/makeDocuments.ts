import type { Authorizer } from "../../ports/authz.ts";
import makeCreateDocument from "./makeCreateDocument.ts";
import makeReadDocument from "./makeReadDocument.ts";
import makeUpdateDocument from "./makeUpdateDocument.ts";
import type { DocumentRepository, Documents } from "./types.ts";

type DocumentsCfg = {
    repository: DocumentRepository;
    authorizer: Authorizer;
};

const makeDocuments = ({ repository, authorizer }: DocumentsCfg): Documents => {
    const create = makeCreateDocument({ repository, authorizer });
    const read   = makeReadDocument({ repository, authorizer });
    const update = makeUpdateDocument({ repository, authorizer });

    return { create, read, update };
};

export default makeDocuments;
