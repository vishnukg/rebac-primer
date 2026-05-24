// Read a document by id.
//
// authz rule: the actor must have `can_read` on the document.
// This is satisfied by being a viewer, editor, or owner — directly or via workspace/team.

import { document } from "../../ports/authz.ts";
import type { Authorizer } from "../../ports/authz.ts";
import type { DocumentRepository, ReadDocumentFn } from "./types.ts";
import { DocumentNotFoundError, ForbiddenError } from "./types.ts";

type ReadDocumentCfg = {
    repository: DocumentRepository;
    authorizer: Authorizer;
};

const makeReadDocument = ({ repository, authorizer }: ReadDocumentCfg): ReadDocumentFn => {
    return async (id, actor) => {
        const existing = await repository.findById(id);
        if (!existing) throw new DocumentNotFoundError(id);

        const decision = await authorizer.check({
            user:     actor,
            relation: "can_read",
            object:   document(id),
        });
        if (!decision.allowed) throw new ForbiddenError(`${actor} cannot read document:${id}`);

        return existing;
    };
};

export default makeReadDocument;
