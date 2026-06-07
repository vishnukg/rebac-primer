import type { TupleRepository } from "../ports/index.ts";
import type { AuthzService } from "./types.ts";

const makeDeleteTuples = ({
    repository,
}: {
    repository: TupleRepository;
}): AuthzService["deleteTuples"] => {
    return async (tuples) => {
        for (const t of tuples) repository.delete(t);
    };
};

export default makeDeleteTuples;
