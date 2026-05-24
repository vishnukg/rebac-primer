import type { DocumentOperationDeps, RequireAllowedFn } from "./types.ts";
import { ForbiddenError } from "./types.ts";

const makeRequireAllowed = ({ authorizer }: Pick<DocumentOperationDeps, "authorizer">): RequireAllowedFn => {
  const requireAllowed: RequireAllowedFn = async (actor, relation, object, action) => {
    const decision = await authorizer.check({ user: actor, relation, object });
    if (!decision.allowed) {
      throw new ForbiddenError(`${actor} cannot ${action} ${object}`);
    }
  };

  return requireAllowed;
};

export default makeRequireAllowed;
