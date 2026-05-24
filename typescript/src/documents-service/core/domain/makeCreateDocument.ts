// Create a document and register its relationships with the authz service.
//
// authz check: actor must be editor (or higher) of the workspace.
//
// After saving the document, two tuples are written to the authz service:
//   (document:id, workspace, workspace:X)  — so the graph knows where this
//                                            document lives (enables inheritance)
//   (document:id, owner, user:actor)        — the creator has direct ownership
//
// This is how the documents service keeps the authz service up to date.
// Product teams never need to write document-level tuples manually.

import { document, tuple } from "../../../shared/rebac.ts";
import type { AuthzClient } from "../ports/authzClient.ts";
import type { DocumentRepository } from "../ports/documentRepository.ts";
import type { CreateDocumentFn } from "./types.ts";
import { ForbiddenError } from "./types.ts";

type Cfg = { repository: DocumentRepository; authzClient: AuthzClient };

const makeCreateDocument = ({ repository, authzClient }: Cfg): CreateDocumentFn =>
    async input => {
        // 1. Authz check — can this actor create in this workspace?
        const { allowed } = await authzClient.check({
            user:     input.actor,
            relation: "editor",
            object:   input.workspace,
        });
        if (!allowed) {
            throw ForbiddenError(`${input.actor} cannot create documents in ${input.workspace}`);
        }

        // 2. Persist the document.
        const doc = {
            id:        input.id,
            title:     input.title,
            body:      input.body,
            workspace: input.workspace,
            updatedBy: input.actor,
        };
        await repository.save(doc);

        // 3. Register relationships in the authz service so future checks work.
        //    This is the write-back pattern: the documents service owns
        //    document-level tuples; the authz service owns workspace/team tuples.
        await authzClient.writeTuples([
            tuple(document(input.id), "workspace", input.workspace),
            tuple(document(input.id), "owner",     input.actor),
        ]);

        return doc;
    };

export default makeCreateDocument;
