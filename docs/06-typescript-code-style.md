# TypeScript code style

This repo values simplicity, readability, and maintainability over cleverness.

Style is not decoration. It is how future readers decide whether code is safe to
change.

## Scene

Imagine opening this repo six months from now. You remember the idea, but not
the details. Good style is what lets you safely make the next change without
relearning the whole system.

## The house style in one page

- Keep modules small and named after the domain concept they own.
- Prefer plain functions, classes, and interfaces before adding libraries.
- Use object-oriented boundaries for services, stores, and adapters.
- Use `type` aliases for data shapes, unions, and computed types.
- Use `interface` for behavior contracts such as `Authorizer`.
- Keep `strict` TypeScript enabled.
- Avoid `as` casts in application code.
- Validate strings at boundaries.
- Keep module imports direct and explicit.
- Prefer explicit composition over hidden singleton state.
- Write tests that describe behavior.
- Prefer readable loops over dense cleverness in teaching code.
- Add abstractions only after duplication becomes distracting.

## What "simple" means here

Simple does not mean under-engineered.

Simple means:

- the domain words are visible
- the control flow is easy to trace
- there are few surprising dependencies
- errors are explicit
- tests describe the rule being protected

This is simple:

```ts
await this.requireAllowed(input.actor, "can_edit", documentObject(input.id), "edit");
```

It says who needs what relation on which object.

This is not simple:

```ts
await policyEngine.evaluate(ctx, Action.DocumentUpdate, Resource.from(input));
```

That style might be fine in a larger system, but it hides the ReBAC vocabulary
this repo is trying to teach.

## Object-oriented style

This repo uses object-oriented structure where it improves maintainability:

- service classes coordinate business actions
- repository classes own persistence state
- authorizer classes implement a shared behavior contract
- private methods keep repeated mechanics local
- constructor injection makes dependencies visible

Good:

```ts
class DocumentService {
  constructor(
    private readonly repository: DocumentRepository,
    private readonly authorizer: Authorizer
  ) {}
}
```

Also good:

```ts
interface Authorizer {
  check(request: CheckRequest): Promise<CheckResult>;
}
```

This gives the code polymorphism without forcing inheritance.

Avoid deep class hierarchies unless the domain truly has stable specialization.
Most backend code is easier to maintain with composition:

```text
DocumentService has an Authorizer
DocumentService has a DocumentRepository
```

Rather than inheritance:

```text
AuthorizedDocumentService extends BaseDocumentService extends ServiceBase
```

Use classes when they own state, dependencies, or meaningful behavior. Use plain
types and functions for small immutable values.

## Naming

Use names that match the domain:

- `TupleKey`
- `Relation`
- `SubjectSet`
- `Authorizer`
- `DocumentService`
- `MemoryTupleStore`

Avoid names that describe implementation mechanics without domain meaning:

- `DataManager`
- `Processor`
- `Helper`
- `Util`
- `Thing`

A good name reduces comments.

## Comments

Most code should not need comments. Prefer names and structure first.

Use comments when:

- a line encodes a non-obvious domain rule
- a workaround exists because of a library or platform limitation
- a short explanation prevents a future bug

Do not comment the obvious:

```ts
// Returns the document
return existing;
```

## Type design

The strongest types in this repo are not complicated. They are specific.

```ts
export type TeamRelation = "member" | "admin";
```

That is better than:

```ts
export type TeamRelation = string;
```

Specific types make illegal states harder to express.

## Runtime validation still matters

TypeScript cannot protect you from a bad string coming from a request, config
file, database, or SDK.

So parsing functions accept raw strings:

```ts
parseObject(value: string)
```

And return typed data only after validation.

That boundary is important. Inside the app, use typed helpers. At the edge,
validate.

## Casts

Treat this as a smell in application code:

```ts
value as RebacObject<"workspace">
```

A cast can be correct, but it bypasses the compiler's skepticism. Prefer a type
guard:

```ts
if (isObjectOfType(parent.user, "workspace")) {
  // parent.user is now RebacObject<"workspace">
}
```

`as const` is different. It is often a good way to preserve literal values:

```ts
export const relationshipGraphExample = [
  "team:platform#member contains user:alice",
  "workspace:acme#editor contains team:platform#member"
] as const;
```

## Errors

Throw domain errors from domain code:

```ts
throw new ForbiddenError(`${actor} cannot edit ${object}`);
```

Do not return booleans for operations where the caller needs to distinguish
between "not found" and "not allowed."

Do not throw strings.

## Tests

Test behavior at the level where the rule matters.

For ReBAC graph logic, test the graph:

```ts
expect(result.allowed).toBe(true);
```

For service logic, test the business action:

```ts
await expect(service.update(input)).rejects.toBeInstanceOf(ForbiddenError);
```

Avoid testing private methods directly. Private methods are implementation
details. Public behavior is the contract.

## ReBAC-specific rule

Every permission should be explainable as a graph path:

```text
user -> relation -> object -> inherited relation -> permission
```

If you cannot explain the path in one or two lines, simplify the model before
adding more code.

## Module-specific rule

Library modules should export capabilities. Entrypoints should perform actions.

Good:

```ts
export class GraphAuthorizer {}
```

Good:

```ts
export const openFgaModel = `...`;
```

Risky:

```ts
export const client = new OpenFgaClient(loadConfigFromEnv());
```

That creates infrastructure as a side effect of importing the module. Prefer a
factory or explicit composition point unless the value is immutable data.

## Review checklist

Before committing a change, ask:

- Does the code use domain names instead of generic names?
- Did I avoid unnecessary casts?
- Did I keep parsing at the boundary?
- Did I add or update tests for the behavior?
- Can I explain the authorization path in plain English?
- Does `npm run build` pass?
- Does `npm test` pass?

## Checkpoint

Before adding an abstraction, ask:

```text
Does this make the ReBAC rule easier to see?
```

If the answer is no, keep the code boring.
