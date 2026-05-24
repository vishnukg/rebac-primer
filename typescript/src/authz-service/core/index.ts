// AuthZ service public core API.
export * from "./domain/types.ts";
export * from "./ports/tupleRepository.ts";
export { default as makeAuthzDomain } from "./domain/makeAuthzDomain.ts";
