import { document } from "../authz/index.ts";
import type { DocumentOperationDeps, RequireAllowedFn, UpdateDocumentFn } from "./types.ts";
import { DocumentNotFoundError } from "./types.ts";

type UpdateDocumentCfg = Pick<DocumentOperationDeps, "repository"> & {
  requireAllowed: RequireAllowedFn;
};

const makeUpdateDocument = ({ repository, requireAllowed }: UpdateDocumentCfg): UpdateDocumentFn => {
  const update: UpdateDocumentFn = async input => {
    const existing = await repository.findById(input.id);
    if (!existing) {
      throw new DocumentNotFoundError(input.id);
    }

    await requireAllowed(input.actor, "can_edit", document(input.id), "edit");

    const updated = { ...existing, body: input.body, updatedBy: input.actor };
    await repository.save(updated);
    return updated;
  };

  return update;
};

export default makeUpdateDocument;
