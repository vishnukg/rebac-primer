import { OpenFgaClient } from "@openfga/sdk";
import type { Authorizer, CheckRequest, CheckResult, TupleKey } from "./types.js";

export type OpenFgaConfig = Readonly<{
  apiUrl: string;
  storeId: string;
  authorizationModelId?: string;
}>;

export class OpenFgaAuthorizer implements Authorizer {
  private readonly client: OpenFgaClient;

  constructor(config: OpenFgaConfig) {
    this.client = new OpenFgaClient(config);
  }

  async check(request: CheckRequest): Promise<CheckResult> {
    const response = await this.client.check({
      user: request.user,
      relation: request.relation,
      object: request.object
    });

    return {
      allowed: response.allowed === true,
      trace: ["OpenFGA evaluated the relationship graph remotely"]
    };
  }

  async writeTuples(tuples: readonly TupleKey[]): Promise<void> {
    await this.client.write({
      writes: tuples.map((tupleKey) => ({
        user: tupleKey.user,
        relation: tupleKey.relation,
        object: tupleKey.object
      }))
    });
  }
}
