# Types and values

This doc is the heart of the TypeScript primer.

TypeScript becomes useful when your types describe the actual domain instead of
decorating JavaScript with generic labels. This project gives the compiler a
vocabulary for ReBAC:

- object ids
- subject sets
- relations
- tuple keys
- authorization checks

Open `src/authz/types.ts` while reading.

## Primitive types are only the start

JavaScript gives you runtime primitives:

```ts
const count = 3;          // number
const name = "alice";     // string
const allowed = true;     // boolean
const missing = undefined;
```

TypeScript can type those, but real leverage comes from domain types:

```ts
type ObjectType = "user" | "team" | "workspace" | "document";
```

`ObjectType` is more meaningful than `string`. It says there are exactly four
object categories in this authorization model.

## Literal unions

A literal union is a small, closed set of allowed values.

```ts
export type TeamRelation = "member" | "admin";
```

This is perfect for authorization because relation names are not casual text.
They are part of the model contract.

Bad:

```ts
function check(relation: string) {}
```

Better:

```ts
function check(relation: Relation) {}
```

Now callers cannot pass `"whatever"` without fighting the compiler.

## Template literal types

OpenFGA object ids have a shape:

```text
type:id
```

This repo models that with a template literal type:

```ts
export type RebacObject<TType extends ObjectType = ObjectType> =
  `${TType}:${string}`;
```

That lets TypeScript distinguish these:

```ts
const alice: RebacObject<"user"> = "user:alice";
const roadmap: RebacObject<"document"> = "document:roadmap";
```

This is not runtime validation. The compiler uses it while checking code.

## Generic type parameters

This part:

```ts
TType extends ObjectType
```

means "`TType` can be any one of the allowed object types."

So `RebacObject<"user">` becomes:

```ts
`user:${string}`
```

And `RebacObject<"document">` becomes:

```ts
`document:${string}`
```

Generics are not automatically advanced. Used carefully, they let one type
represent a family of related shapes.

## Helper functions improve readability

You could write this everywhere:

```ts
const alice = "user:alice" as RebacObject<"user">;
```

Do not make that the normal style. A cast tells TypeScript, "trust me." That is
a useful escape hatch at boundaries, but a poor habit in application code.

This repo uses helper functions:

```ts
export function user(id: string): RebacObject<"user"> {
  return object("user", id);
}
```

Now calling code is readable:

```ts
const alice = user("alice");
const roadmap = document("roadmap");
```

The helper also validates empty ids at runtime.

## `type` vs `interface`

Use `type` when you are naming data shapes, unions, or computed types:

```ts
export type CheckRequest = Readonly<{
  user: RebacObject<"user">;
  relation: Relation;
  object: RebacObject;
}>;
```

Use `interface` when you are describing behavior:

```ts
export interface Authorizer {
  check(request: CheckRequest): Promise<CheckResult>;
}
```

This is not a universal law, but it is a clean local convention.

## Narrowing

Narrowing means TypeScript starts with a broad type and refines it after a check.

Example from this repo:

```ts
export function isSubjectSet(subject: Subject): subject is SubjectSet {
  return subject.includes("#");
}
```

The return type `subject is SubjectSet` is a type predicate. It tells the
compiler that after this function returns `true`, the variable is a subject set.

That makes this code safe:

```ts
if (isSubjectSet(tupleKey.user)) {
  parseSubjectSet(tupleKey.user);
}
```

Inside the `if`, TypeScript knows `tupleKey.user` is no longer a plain object id.

## Parsing is a boundary

Types are erased at runtime, so any string that comes from outside your code
needs validation.

```ts
export function parseObject(value: string): { type: ObjectType; id: string } {
  const [type, ...idParts] = value.split(":");

  if (!isObjectType(type) || idParts.length === 0 || idParts.join(":").length === 0) {
    throw new Error(`Invalid OpenFGA object id: ${value}`);
  }

  return { type, id: idParts.join(":") };
}
```

Notice the shape:

1. accept a raw `string`
2. inspect it
3. either throw or return a typed result

That is cleaner than pretending every string is already safe.

## A maintainability rule

If you see this in normal application code:

```ts
value as SomeImportantType
```

pause. Ask whether a type guard, parser, or helper function would make the code
clearer.

There are legitimate casts. `as const` is often fine because it preserves literal
types:

```ts
const graph = ["user:alice", "document:roadmap"] as const;
```

But casts that silence uncertainty should be rare.

## Exercise

Add a new document permission:

```ts
"can_archive"
```

Then:

1. add it to `DocumentRelation`
2. decide which relationship should imply it
3. update `GraphAuthorizer`
4. add a test
5. run `npm run build && npm test`

This exercise teaches the best part of TypeScript: when the vocabulary changes,
the compiler helps you find the edges of the change.
