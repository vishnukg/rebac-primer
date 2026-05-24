import type { Authenticator } from "../authn/index.ts";
import type { Documents } from "../documents/index.ts";
import type { JsonObject } from "./json.ts";

export type HttpRequest = {
  method:        string;
  path:          string;
  query:         URLSearchParams;
  authorization: string | undefined;
  body?:         unknown;
};

export type HttpResponse = {
  statusCode: number;
  body:       JsonObject;
};

export type HttpHandler = (request: HttpRequest) => Promise<HttpResponse>;

export type HttpHandlerCfg = {
  authenticator: Authenticator;
  documents:     Documents;
};
