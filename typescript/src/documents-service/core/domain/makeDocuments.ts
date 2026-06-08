// The documents domain — one factory that closes over its driven ports
// (the repository and the authz client) and exposes the three operations as
// methods. This is the module pattern at its core: dependencies hidden in the
// closure, behaviour exposed as a noun (Documents) whose methods are verbs
// (create, read, update).
//
// One factory, one noun: the domain IS the Documents port, so its operations
// live inside it as methods rather than as three standalone make* factories
// that then have to be re-bundled. It calls no other factory, so it is a leaf
// (a make*, not a compose*) — mirrors makeRestaurant in the ModulePattern repo
// and makeAuthzService on the authz side.

import { document, tuple } from "../../../shared/rebac.ts";
import type { AuthzClient } from "../ports/authzClient.ts";
import type { DocumentRepository } from "../ports/documentRepository.ts";
import type { Documents } from "./types.ts";
import { DocumentNotFoundError, ForbiddenError } from "./types.ts";

type MakeDocumentsCfg = {
    repository:  DocumentRepository;
    authzClient: AuthzClient;
};

const makeDocuments = ({ repository, authzClient }: MakeDocumentsCfg): Documents => {
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
    const create: Documents["create"] = async input => {
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

    // Read returns a document if the actor has can_read access.
    //
    // Existence is checked before authorization so the error is accurate: a missing
    // document throws not-found, not forbidden.
    //
    // Security tradeoff: this ordering leaks existence. A denied actor gets 403 for a
    // document that exists but 404 for one that does not, so they can probe which ids
    // exist even without access. That is fine for this tutorial — clear errors aid
    // learning — but high-security systems return 404 for both cases so the two are
    // indistinguishable (check authorization first, then map a denial to not-found).
    // See docs/40-production-readiness.md (Gap 13).
    const read: Documents["read"] = async ({ id, actor }) => {
        const doc = await repository.findById(id);
        if (!doc) throw DocumentNotFoundError(id);

        const { allowed } = await authzClient.check({
            user:     actor,
            relation: "can_read",
            object:   document(id),
        });
        if (!allowed) throw ForbiddenError(`${actor} cannot read ${id}`);

        return doc;
    };

    // Update saves new body text if the actor has can_edit access. Existence is
    // checked before authorization, for the same reason as read above.
    const update: Documents["update"] = async ({ id, body, actor }) => {
        const existing = await repository.findById(id);
        if (!existing) throw DocumentNotFoundError(id);

        const { allowed } = await authzClient.check({
            user:     actor,
            relation: "can_edit",
            object:   document(id),
        });
        if (!allowed) throw ForbiddenError(`${actor} cannot edit ${id}`);

        const updated = { ...existing, body, updatedBy: actor };
        await repository.save(updated);
        return updated;
    };

    return { create, read, update };
};

export default makeDocuments;
