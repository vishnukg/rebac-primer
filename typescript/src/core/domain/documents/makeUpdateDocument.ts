// Update a document's body.
//
// authz rule: the actor must have `can_edit` on the document.
// This is satisfied by being an editor or owner — directly or via workspace/team.

import { document } from "../../ports/authz.ts";
import type { Authorizer } from "../../ports/authz.ts";
import type { DocumentRepository, UpdateDocumentFn } from "./types.ts";
import { DocumentNotFoundError, ForbiddenError } from "./types.ts";

type UpdateDocumentCfg = {
    repository: DocumentRepository;
    authorizer: Authorizer;
};

const makeUpdateDocument = ({ repository, authorizer }: UpdateDocumentCfg): UpdateDocumentFn => {
    return async input => {
        const existing = await repository.findById(input.id);
        if (!existing) throw new DocumentNotFoundError(input.id);

        const decision = await authorizer.check({
            user:     input.actor,
            relation: "can_edit",
            object:   document(input.id),
        });
        if (!decision.allowed) {
            throw new ForbiddenError(`${input.actor} cannot edit document:${input.id}`);
        }

        const updated = { ...existing, body: input.body, updatedBy: input.actor };
        await repository.save(updated);
        return updated;
    };
};

export default makeUpdateDocument;
