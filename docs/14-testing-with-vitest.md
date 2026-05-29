# Testing TypeScript with Vitest

Tests are where TypeScript code stops being a pile of plausible types and starts
proving behavior.

## Scene

Alice, Bob, and Casey are your test audience. Alice should edit. Bob should
read but fail to edit. Casey should be denied. If the tests can prove those
three stories, they are doing more than checking lines of code.

This repo uses Vitest because it is fast, TypeScript-friendly, and familiar if
you have seen Jest-style tests.

Run:

```bash
npm test
```

Run with coverage:

```bash
npm run coverage
```

## What tests should prove

Authorization tests should be concrete. A test that only proves "the mock was
called" does not prove the graph grants or denies the right access.

The repo organizes tests around layers — types, model text, ReBAC traversal,
domain rules, HTTP mapping, client behavior, and infrastructure adapters:

| Test file | What it teaches |
|-----------|-----------------|
| `test/graphEvaluator.test.ts` | graph traversal through the ReBAC engine |
| `test/authz.test.ts` | shared `rebac.ts` helper constructors and parsers |
| `test/documents.test.ts` | business actions enforce authorization |
| `test/authzService.test.ts` | AuthZ service HTTP mapping without opening sockets |
| `test/documentsService.test.ts` | Documents service HTTP mapping without opening sockets |
| `test/client.test.ts` | client behavior with injected dependencies |
| `test/authn.test.ts` | bearer-token verification behavior |
| `test/repository.test.ts` | document repository copy semantics |

The tests double as executable documentation.

## Anatomy of a Vitest test

```ts
import { describe, expect, it } from "vitest";

describe("makeGraphEvaluator", () => {
  it("allows alice to edit via team → workspace → document chain", async () => {
    // Arrange
    const repository = makeInMemoryTupleRepository({ seed: [...seedPolicyTuples(), docWorkspaceTuple] });
    const evaluator  = makeGraphEvaluator({ repository });

    // Act
    const { allowed } = await evaluator.evaluate({
      user: alice,
      relation: "can_edit",
      object: roadmapDoc,
    });

    // Assert
    expect(allowed).toBe(true);
  });
});
```

Read the outer `describe` as the subject under test. Read `it` as a behavior
sentence.

Good test names are not cute. They say what behavior matters.

## Test naming convention

Every unit test should state the behavior in plain language:

```text
allows alice to edit through team membership and workspace inheritance
```

Examples:

```ts
it("lets bob read as a workspace viewer but denies editing", async () => {});
```

```ts
it("returns 403 when ReBAC denies the action", async () => {});
```

Test output should read like a behavior list.

## Arrange, act, assert

Most tests in this repo follow this shape:

```text
arrange: create tuples, stores, services
act: call the method being tested
assert: check the observable result
```

Example:

```ts
// Arrange
const repository = makeInMemoryTupleRepository({ seed: [...seedPolicyTuples(), docWorkspaceTuple] });
const evaluator  = makeGraphEvaluator({ repository });

// Act
const { allowed } = await evaluator.evaluate({
  user: bob,
  relation: "can_edit",
  object: roadmapDoc,
});

// Assert
expect(allowed).toBe(false);
```

This is small, but it tells a story:

- Bob exists.
- Bob asks to edit the roadmap document.
- The graph denies him.

That is more valuable than a test that asserts an internal method was called.

## Small shared setup is okay

Tests in this repo keep setup close to the behavior being tested.

Small local helpers are fine when they remove noise without hiding the important
relationships:

```ts
const makeDocumentService = () => {
  const repository  = makeInMemoryDocumentRepository();
  const authzClient = makeInProcessAuthzClient(seedPolicyTuples());
  return makeDocuments({ repository, authzClient });
};
```

Production fixtures such as `seedPolicyTuples()` are allowed because they are part
of the lesson data, not a hidden test helper.

The current convention is also enforced by review:

- test names describe behavior
- tests use visible Arrange / Act / Assert sections
- socket and TUI entrypoints are excluded from coverage; their core logic is
  tested behind interfaces

## Testing async code

Vitest works naturally with `async` tests:

```ts
it("rejects creation for workspace viewers", async () => {
  // Arrange
  const documents = makeDocumentService();

  // Act
  const createPromise = documents.create({
    id: "incident-plan",
    title: "Incident Plan",
    body: "Draft",
    workspace: productWorkspace,
    actor: bob
  });

  // Assert
  await expect(createPromise).rejects.toMatchObject({ name: "ForbiddenError" });
});
```

Two details matter:

- return or `await` the expectation
- assert the domain error, not a vague failure

## Test data should be readable

Open `test/fixtures.ts`.

```ts
export const seedPolicyTuples = (): TupleKey[] => [
  tuple(platformTeam, "member", alice),
  tuple(productWorkspace, "editor", subjectSet(platformTeam, "member")),
  tuple(productWorkspace, "viewer", bob),
];
```

This fixture is intentionally tiny. You can hold the whole graph in your head.

Avoid giant fixtures unless the test truly needs them. Big fixtures make tests
look realistic while hiding the one relationship that matters.

## When to use mocks

Mocks are useful when you want to isolate a unit.

For example, a `DenyAllEvaluator` can prove the service rejects unauthorized
creates without caring about graph traversal.

But this repo mostly uses `makeGraphEvaluator` (via `makeInProcessAuthzClient`)
because the tutorial goal is to learn ReBAC. In that context, graph behavior is
not an implementation detail. It is the lesson.

## What not to test

Avoid tests like:

```ts
expect(authzClient.check).toHaveBeenCalled();
```

That proves a call happened, not that access control is correct.

Prefer:

```ts
await expect(
  evaluator.evaluate({ user: alice, relation: "can_edit", object: roadmapDoc })
).resolves.toMatchObject({ allowed: true });
```

Now the test states the rule.

## Coverage is a signal, not the goal

Coverage can show you untested areas. It cannot tell you whether your tests are
good.

For authorization, the most important cases are:

- direct access allowed
- inherited access allowed
- subject-set access allowed
- near-miss denied
- unrelated user denied
- service method rejects denied action

The "near-miss" case is especially important. Bob can read the roadmap
document, but cannot edit it. That proves the graph is not simply letting
everyone through.

## Exercise

Add a new test for this rule:

```text
workspace viewers like Bob can comment on documents but cannot edit them
```

Use `can_comment` and `can_edit` against `bob`.

Before running the test, predict the result from the tuples:

```text
workspace:productWorkspace viewer user:bob
document:roadmapDocument workspace workspace:productWorkspace
document can_comment = viewer
document can_edit = editor
```

Then run:

```bash
npm test
```

## Checkpoint

Why is this a better authorization assertion:

```ts
expect(result.allowed).toBe(false);
```

than this:

```ts
expect(authzClient.check).toHaveBeenCalled();
```

Good answer: the first checks the rule. The second only checks that a method was
called.
