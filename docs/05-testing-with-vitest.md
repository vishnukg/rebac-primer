# Testing TypeScript with Vitest

Tests are where TypeScript code stops being a pile of plausible types and starts
proving behavior.

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

This repo uses four styles:

| Test file | What it teaches |
|-----------|-----------------|
| `test/types-and-store.test.ts` | helper functions and tuple storage |
| `test/model.test.ts` | model text contains required relationships |
| `test/graph-authorizer.test.ts` | ReBAC traversal behavior |
| `test/document-service.test.ts` | business actions enforce authorization |

The tests double as executable documentation.

## Anatomy of a Vitest test

```ts
import { describe, expect, it } from "vitest";

describe("GraphAuthorizer", () => {
  it("allows a team member to edit a document through workspace inheritance", async () => {
    const authorizer = new GraphAuthorizer(new MemoryTupleStore(tutorialTuples()));

    const result = await authorizer.check({
      user: alice,
      relation: "can_edit",
      object: roadmap
    });

    expect(result.allowed).toBe(true);
  });
});
```

Read the outer `describe` as the subject under test. Read `it` as a behavior
sentence.

Good test names are not cute. They say what behavior matters.

## Arrange, act, assert

Most tests in this repo follow this shape:

```text
arrange: create tuples, stores, services
act: call the method being tested
assert: check the observable result
```

Example:

```ts
const authorizer = new GraphAuthorizer(new MemoryTupleStore(tutorialTuples()));

const result = await authorizer.check({
  user: bob,
  relation: "can_edit",
  object: roadmap
});

expect(result.allowed).toBe(false);
```

This is small, but it tells a story:

- Bob exists.
- Bob asks to edit the roadmap.
- The graph denies him.

That is more valuable than a test that asserts an internal method was called.

## Testing async code

Vitest works naturally with `async` tests:

```ts
it("rejects creates when the actor has no workspace editor path", async () => {
  const service = serviceWithTuples(tutorialTuples());

  await expect(
    service.create({
      id: "incident-plan",
      title: "Incident Plan",
      body: "Draft",
      workspace: acme,
      actor: bob
    })
  ).rejects.toBeInstanceOf(ForbiddenError);
});
```

Two details matter:

- return or `await` the expectation
- assert the domain error, not a vague failure

## Test data should be readable

Open `src/testing/fixtures.ts`.

```ts
export function tutorialTuples(): readonly TupleKey[] {
  return [
    tuple(platform, "member", alice),
    tuple(acme, "editor", subjectSet(platform, "member")),
    tuple(acme, "viewer", bob),
    tuple(roadmap, "workspace", acme)
  ];
}
```

This fixture is intentionally tiny. You can hold the whole graph in your head.

Avoid giant fixtures unless the test truly needs them. Big fixtures make tests
look realistic while hiding the one relationship that matters.

## When to use mocks

Mocks are useful when you want to isolate a unit.

For example, a `DenyAllAuthorizer` can prove the service rejects unauthorized
creates without caring about graph traversal.

But this repo mostly uses the in-memory `GraphAuthorizer` because the tutorial
goal is to learn ReBAC. In that context, graph behavior is not an implementation
detail. It is the lesson.

## What not to test

Avoid tests like:

```ts
expect(authorizer.check).toHaveBeenCalled();
```

That proves a call happened, not that access control is correct.

Prefer:

```ts
await expect(
  authorizer.check({ user: alice, relation: "can_edit", object: roadmap })
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

The "near-miss" case is especially important. Bob can read the roadmap, but he
cannot edit it. That proves the graph is not simply letting everyone through.

## Exercise

Add a new test for this rule:

```text
workspace viewers can comment on documents but cannot edit them
```

Use `can_comment` and `can_edit` against `bob`.

Before running the test, predict the result from the tuples:

```text
workspace:acme viewer user:bob
document:roadmap workspace workspace:acme
document can_comment = viewer
document can_edit = editor
```

Then run:

```bash
npm test
```
