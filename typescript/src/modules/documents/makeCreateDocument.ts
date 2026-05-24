import type { CreateDocumentFn, DocumentOperationDeps, RequireAllowedFn } from "./types.ts";

type CreateDocumentCfg = Pick<DocumentOperationDeps, "repository"> & {
  requireAllowed: RequireAllowedFn;
};

const makeCreateDocument = ({ repository, requireAllowed }: CreateDocumentCfg): CreateDocumentFn => {
  const create: CreateDocumentFn = async input => {
    await requireAllowed(input.actor, "editor", input.workspace, "create documents in");

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

  return create;
};

export default makeCreateDocument;
