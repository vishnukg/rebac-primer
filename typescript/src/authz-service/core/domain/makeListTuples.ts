import type { TupleRepository } from "../ports/index.ts";
import type { AuthzService } from "./types.ts";

// Pure pass-through to the repository today — reading tuples adds no domain logic.
// Wrapped in a factory anyway so every domain operation is built the same way and
// there's one place to add filtering/paging/audit later.
const makeListTuples = ({
    repository,
}: {
    repository: TupleRepository;
}): AuthzService["listTuples"] => {
    return async (filter) => repository.findAll(filter);
};

export default makeListTuples;
