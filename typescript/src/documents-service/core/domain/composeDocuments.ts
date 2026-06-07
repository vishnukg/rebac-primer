// Composes the documents domain: builds the create/read/update operations from
// the repository and authz client, and returns them as the Documents port.
//
// This is a compose* (not a make*) because it builds its own collaborators via
// make* factories — the same role composeRestaurant plays in the ModulePattern
// reference repo. makeAuthzDomain, by contrast, defines its operations inline
// and so stays a make*.

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

const composeDocuments = ({ repository, authzClient }: DocumentsCfg): Documents => {
    const create = makeCreateDocument({ repository, authzClient });
    const read   = makeReadDocument({ repository, authzClient });
    const update = makeUpdateDocument({ repository, authzClient });
    return { create, read, update };
};

export default composeDocuments;
