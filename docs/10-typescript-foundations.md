# TypeScript foundations

TypeScript is JavaScript with a type system and a compiler.

That sounds small, but it changes how you work. In plain JavaScript, many
mistakes are found only when code runs. In TypeScript, a lot of those mistakes
are found while you are still editing.

This matters for authorization code. A typo like `can_edti` should not wait
until production traffic proves it is wrong.

## Scene

You have inherited a permission system. The code "works" until somebody ships a
misspelled relation and a real user gets denied. Your first job is to make those
mistakes loud while you are still editing.

By the end of this chapter, you should be able to answer:

- what TypeScript checks
- what still happens at runtime
- why `strict` mode is worth the friction
- how a tiny typo becomes a compiler error instead of a production incident

## The mental model

Think of TypeScript as three layers:

```text
your .ts files
  -> TypeScript compiler checks the code
  -> JavaScript runs in Node or the browser
```

Types do not exist at runtime. They are design-time guardrails.

This compiles:

```ts
const userId: string = "alice";
```

At runtime, Node only sees JavaScript:

```js
const userId = "alice";
```

So TypeScript is not a runtime permission system. It cannot stop an attacker by
itself. Its job is to make your program harder to write incorrectly.

## Why this repo uses strict TypeScript

Open `tsconfig.json`.

The important options are:

```json
{
  "strict": true,
  "noImplicitReturns": true,
  "noUncheckedIndexedAccess": true,
  "exactOptionalPropertyTypes": true,
  "noUnusedLocals": true,
  "noUnusedParameters": true
}
```

These settings are not ceremony. They support maintainability:

- `strict` turns on the type checks that make TypeScript worth using.
- `noImplicitReturns` catches functions that accidentally forget a branch.
- `noUncheckedIndexedAccess` reminds you that array/map lookups can miss.
- `exactOptionalPropertyTypes` makes optional fields behave honestly.
- `noUnusedLocals` and `noUnusedParameters` keep dead code out of lessons.

When code is educational, unused code is extra harmful. Learners assume every
line matters.

## Values vs types

JavaScript has values:

```ts
const relation = "can_edit";
```

TypeScript can infer a type for that value:

```ts
// relation is the literal type "can_edit"
const relation = "can_edit";
```

But with `let`, TypeScript widens the type because the value can change:

```ts
let relation = "can_edit";
// relation is string
```

That difference matters. The authorization model has a closed vocabulary. We
want `can_edit`, `can_read`, and `can_delete`, not any random string.

## Your first useful type

Open `src/shared/rebac.ts`.

```ts
export type DocumentRelation =
  | "workspace"
  | "owner"
  | "editor"
  | "viewer"
  | "can_read"
  | "can_comment"
  | "can_edit"
  | "can_delete";
```

This is a union type. It says a document relation must be one of these exact
strings.

That gives you a better failure mode:

```ts
const relation: DocumentRelation = "can_edti";
```

The compiler rejects it before the code runs.

## TypeScript should encode domain language

Weak version:

```ts
type Tuple = {
  object: string;
  relation: string;
  user: string;
};
```

This is technically typed, but it does not teach the compiler anything useful.
Everything important is still "just a string."

Better version from this repo:

```ts
export type TupleKey = Readonly<{
  user: Subject;
  relation: Relation;
  object: RebacObject;
}>;
```

Now the code says what the fields mean. A future reader does not need to guess.

## `Readonly` and intent

Tuples are facts. Once created, the clean mental model is that they do not
change. If a relationship changes, write or delete a tuple.

That is why `TupleKey` is `Readonly`.

```ts
const owner = tuple(document("roadmapDocument"), "owner", user("alice"));
owner.relation = "viewer"; // compiler error
```

Immutability keeps examples honest. It also makes tests easier to reason about.

## Modules

Every `.ts` file in this repo is an ES module because `package.json` includes:

```json
{
  "type": "module"
}
```

This repo runs TypeScript source directly in development with `tsx`, so relative
imports include `.ts`:

```ts
import makeGraphEvaluator from "./adapters/authz/makeGraphEvaluator.ts";
```

The important rule is that ESM imports use explicit file extensions. This repo
also sets `allowImportingTsExtensions` because it type-checks source files
without emitting JavaScript from `tsc`.

## Build vs test

Run:

```bash
npm run build
```

This runs the TypeScript compiler. In this repo `tsc` checks types only; the
dev/test commands execute the `.ts` files directly.

Run:

```bash
npm test
```

This runs Vitest. It checks behavior.

You need both. Types prove that the code is structurally coherent. Tests prove
that the program does what the domain requires.

## Exercise

1. Open `test/fixtures.ts`.
2. Change `"editor"` to `"edtor"` in one tuple.
3. Run `npm run build`.
4. Read the compiler error.
5. Restore the code.

The point is not the typo. The point is that the type system understands your
authorization vocabulary.

## Checkpoint

Answer this without looking back:

```text
If TypeScript types disappear at runtime, why are they still useful for ReBAC?
```

Good answer: because they make the authorization vocabulary explicit while
writing code, so invalid relation names and confused object shapes are caught
before the app runs.
