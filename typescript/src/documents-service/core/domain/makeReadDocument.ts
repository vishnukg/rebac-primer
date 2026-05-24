import { document } from "../../../shared/rebac.ts";
import type { AuthzClient } from "../ports/authzClient.ts";
import type { DocumentRepository } from "../ports/documentRepository.ts";
import type { ReadDocumentFn } from "./types.ts";
import { DocumentNotFoundError, ForbiddenError } from "./types.ts";

type ReadDocumentCfg = {
    repository:  DocumentRepository;
    authzClient: AuthzClient;
};

const makeReadDocument = ({ repository, authzClient }: ReadDocumentCfg) => {
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

    return { read };
};

export default makeReadDocument;
