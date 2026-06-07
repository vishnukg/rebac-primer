import type { TupleRepository } from "../ports/index.ts";
import type { AuthzService } from "./types.ts";

const makeWriteTuples = ({
    repository,
}: {
    repository: TupleRepository;
}): AuthzService["writeTuples"] => {
    return async (tuples) => {
        for (const t of tuples) repository.write(t);
    };
};

export default makeWriteTuples;
