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

import { document } from "../../../shared/rebac.ts";
import type { AuthzClient } from "../ports/authzClient.ts";
import type { DocumentRepository } from "../ports/documentRepository.ts";
import type { ReadDocumentFn } from "./types.ts";
import { DocumentNotFoundError, ForbiddenError } from "./types.ts";

type ReadDocumentCfg = {
    repository:  DocumentRepository;
    authzClient: AuthzClient;
};

const makeReadDocument = ({ repository, authzClient }: ReadDocumentCfg): ReadDocumentFn => {
    const read: ReadDocumentFn = async ({ id, actor }) => {
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

    return read;
};

export default makeReadDocument;
