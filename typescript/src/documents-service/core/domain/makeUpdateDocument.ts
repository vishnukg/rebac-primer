import { document } from "../../../shared/rebac.ts";
import type { AuthzClient } from "../ports/authzClient.ts";
import type { DocumentRepository } from "../ports/documentRepository.ts";
import type { UpdateDocumentFn } from "./types.ts";
import { DocumentNotFoundError, ForbiddenError } from "./types.ts";

type UpdateDocumentCfg = {
    repository:  DocumentRepository;
    authzClient: AuthzClient;
};

const makeUpdateDocument = ({ repository, authzClient }: UpdateDocumentCfg) => {
    const update: UpdateDocumentFn = async ({ id, body, actor }) => {
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

    return { update };
};

export default makeUpdateDocument;
