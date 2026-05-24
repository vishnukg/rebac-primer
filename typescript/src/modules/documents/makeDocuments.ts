import type { CreateDocumentFn, Documents, ReadDocumentFn, UpdateDocumentFn } from "./types.ts";

type DocumentsCfg = {
  create: CreateDocumentFn;
  read:   ReadDocumentFn;
  update: UpdateDocumentFn;
};

const makeDocuments = ({ create, read, update }: DocumentsCfg): Documents => ({
  create,
  read,
  update,
});

export default makeDocuments;
