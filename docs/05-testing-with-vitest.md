# Testing TypeScript with Vitest

Tests are where TypeScript code stops being a pile of plausible types and starts
proving behavior.

## Scene

The workspace editor, the workspace viewer, and the outside collaborator are your test
audience. The workspace editor should edit. The workspace viewer should read but fail
to edit. The outside collaborator should be denied. If the tests can prove those three
stories, they are doing more than checking lines of code.

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
| `test/http-handler.test.ts` | HTTP mapping without opening sockets |
| `test/api-client.test.ts` | client behavior with an injected fetcher |
| `test/openfga-client.test.ts` | SDK adapter behavior at the infrastructure boundary |

The tests double as executable documentation.

## Anatomy of a Vitest test

```ts
import { describe, expect, it } from "vitest";

describe("GraphAuthorizer", () => {
  it("given_team_member_workspace_editor_when_checking_document_edit_then_access_is_allowed", async () => {
    // Arrange
    const authorizer = new GraphAuthorizer(new InMemoryTupleStore(seedRelationshipTuples()));

    // Act
    const result = await authorizer.check({
      user: workspaceEditor,
      relation: "can_edit",
      object: roadmapDocument
    });

    // Assert
    expect(result.allowed).toBe(true);
  });
});
```

Read the outer `describe` as the subject under test. Read `it` as a behavior
sentence.

Good test names are not cute. They say what behavior matters.

## Test naming convention

Every unit test should use this name shape:

```text
given_<starting_state>_when_<action>_then_<expected_result>
```

Examples:

```ts
it("given_workspace_viewer_when_checking_document_permissions_then_read_is_allowed_and_edit_is_denied", async () => {});
```

```ts
it("given_missing_document_when_updating_then_not_found_error_is_thrown", async () => {});
```

This convention is intentionally a little verbose. It makes test output read
like a behavior list.

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
const authorizer = new GraphAuthorizer(new InMemoryTupleStore(seedRelationshipTuples()));

// Act
const result = await authorizer.check({
  user: workspaceViewer,
  relation: "can_edit",
  object: roadmapDocument
});

// Assert
expect(result.allowed).toBe(false);
```

This is small, but it tells a story:

- The workspace viewer exists.
- The workspace viewer asks to edit the roadmap document.
- The graph denies him.

That is more valuable than a test that asserts an internal method was called.

## No shared test helper methods

Tests in this repo should keep setup inside the test body.

Avoid local helpers like:

```ts
function serviceWithTuples() {}
```

The repetition is acceptable because this is a teaching repo. A reader should
see the whole setup, action, and assertion without jumping around the file.

Production fixtures such as `seedRelationshipTuples()` are allowed because they are part
of the lesson data, not a hidden test helper.

The current convention is also enforced by review:

- test names use `given_when_then`
- tests use visible Arrange / Act / Assert sections
- test files do not define local setup helper functions
- socket and TUI entrypoints are excluded from coverage; their core logic is
  tested behind interfaces

## Testing async code

Vitest works naturally with `async` tests:

```ts
it("given_workspace_viewer_when_creating_document_then_forbidden_error_is_thrown", async () => {
  // Arrange
  const store = new InMemoryTupleStore(seedRelationshipTuples());
  const service = new DocumentService(
    new InMemoryDocumentRepository(),
    new GraphAuthorizer(store)
  );

  // Act
  const createPromise = service.create({
    id: "incident-plan",
    title: "Incident Plan",
    body: "Draft",
    workspace: productWorkspace,
    actor: workspaceViewer
  });

  // Assert
  await expect(createPromise).rejects.toBeInstanceOf(ForbiddenError);
});
```

Two details matter:

- return or `await` the expectation
- assert the domain error, not a vague failure

## Test data should be readable

Open `src/testing/fixtures.ts`.

```ts
export function seedRelationshipTuples(): readonly TupleKey[] {
  return [
    tuple(platformTeam, "member", workspaceEditor),
    tuple(productWorkspace, "editor", subjectSet(platformTeam, "member")),
    tuple(productWorkspace, "viewer", workspaceViewer),
    tuple(roadmapDocument, "workspace", productWorkspace)
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
  authorizer.check({ user: workspaceEditor, relation: "can_edit", object: roadmapDocument })
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

The "near-miss" case is especially important. The workspace viewer can read the roadmap
document, but cannot edit it. That proves the graph is not simply letting everyone through.

## Exercise

Add a new test for this rule:

```text
workspace viewers can comment on documents but cannot edit them
```

Use `can_comment` and `can_edit` against `workspaceViewer`.

Before running the test, predict the result from the tuples:

```text
workspace:productWorkspace viewer user:workspaceViewer
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
expect(authorizer.check).toHaveBeenCalled();
```

Good answer: the first checks the rule. The second only checks that a method was
called.
