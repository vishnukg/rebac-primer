// Create a document inside a workspace.
//
// authz rule: the actor must be an `editor` (or higher) of the workspace.
// Workspace editors and owners can create; workspace viewers cannot.

import type { Authorizer } from "../../ports/authz.ts";
import type { CreateDocumentFn, DocumentRepository } from "./types.ts";
import { ForbiddenError } from "./types.ts";

type CreateDocumentCfg = {
    repository: DocumentRepository;
    authorizer: Authorizer;
};

const makeCreateDocument = ({ repository, authorizer }: CreateDocumentCfg): CreateDocumentFn => {
    return async input => {
        const decision = await authorizer.check({
            user:     input.actor,
            relation: "editor",
            object:   input.workspace,
        });
        if (!decision.allowed) {
            throw new ForbiddenError(
                `${input.actor} cannot create documents in ${input.workspace}`,
            );
        }

        const created = {
            id:        input.id,
            title:     input.title,
            body:      input.body,
            workspace: input.workspace,
            updatedBy: input.actor,
        };
        await repository.save(created);
        return created;
    };
};

export default makeCreateDocument;
