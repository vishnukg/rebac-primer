import { document } from "../authz/index.ts";
import type { DocumentOperationDeps, ReadDocumentFn, RequireAllowedFn } from "./types.ts";
import { DocumentNotFoundError } from "./types.ts";

type ReadDocumentCfg = Pick<DocumentOperationDeps, "repository"> & {
  requireAllowed: RequireAllowedFn;
};

const makeReadDocument = ({ repository, requireAllowed }: ReadDocumentCfg): ReadDocumentFn => {
  const read: ReadDocumentFn = async (id, actor) => {
    const existing = await repository.findById(id);
    if (!existing) {
      throw new DocumentNotFoundError(id);
    }

    await requireAllowed(actor, "can_read", document(id), "read");
    return existing;
  };

  return read;
};

export default makeReadDocument;
