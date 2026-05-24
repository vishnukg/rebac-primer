# TypeScript AuthZ adapter pattern

The document domain does not talk to the AuthZ service directly.

Instead, it depends on a small interface:

```ts
export interface AuthzClient {
    check:       (request: CheckRequest) => Promise<CheckResult>;
    writeTuples: (tuples: TupleKey[]) => Promise<void>;
}
```

That one interface is the boundary between business logic and authorization
infrastructure.

## Scene

The documents service needs one answer: allowed or denied. The AuthZ service
has its own HTTP API, response shapes, error formats, and network latency. The
adapter keeps those details from spilling into the document domain.

## Why the interface matters

The document use cases should not know about:

- AuthZ service URLs
- HTTP request/response shapes
- JSON parsing
- network retries
- in-process vs remote evaluation

It should know the business rule:

```text
to update a document, the actor must have can_edit on that document
```

That rule appears in code as:

```ts
const { allowed } = await authzClient.check({
    user:     input.actor,
    relation: "can_edit",
    object:   document(input.id),
});
```

The implementation behind `AuthzClient` can change without rewriting the domain.

Architecture:

```text
┌──────────────────────────┐
│  Documents domain        │
│  business rules          │
└───────────┬──────────────┘
            │ depends on interface
            ▼
┌──────────────────────────┐
│  AuthzClient             │
│  check(...)              │
│  writeTuples(...)        │
└────────────┬─────────────┘
             │
    ┌────────┴────────────┐
    ▼                     ▼
┌───────────────┐  ┌─────────────────────┐
│ In-process    │  │ HTTP adapter        │
│ (tests only)  │  │ makeAuthzServiceClient │
└───────────────┘  └──────────┬──────────┘
                              │
                              ▼
                   ┌─────────────────────┐
                   │ AuthZ service :4100 │
                   │ makeGraphEvaluator  │
                   └─────────────────────┘
```

This is composition through interfaces. The document domain does not change when
you swap the `AuthzClient` implementation.

## Two implementations

This repo has two implementations of `AuthzClient`:

```text
makeAuthzServiceClient   -> HTTP adapter for the real AuthZ service (production path)
makeInProcessAuthzClient -> in-process stub used in domain tests
```

They share the same interface. That is why domain tests run fast without a
server, while the running services use real HTTP.

## The HTTP adapter

Open `src/documents-service/adapters/authz/makeAuthzServiceClient.ts`.

```ts
const makeAuthzServiceClient = ({
    baseUrl,
    fetcher = fetch,
}: AuthzServiceClientCfg): AuthzClient => {
    const check = async (request: CheckRequest): Promise<CheckResult> => {
        const response = await fetcher(new URL("/check", baseUrl), {
            method:  "POST",
            headers: { "content-type": "application/json" },
            body:    JSON.stringify(request),
        });
        const json = await response.json();
        return { allowed: json.allowed === true };
    };
    // ...
};
```

The adapter owns the HTTP details. The document domain owns nothing about
how the request travels over the wire.

## The `check` conversion

The adapter converts from this repo's request shape:

```ts
// src/shared/rebac.ts
export type CheckRequest = {
    user:     RebacObject<"user">;
    relation: Relation;
    object:   RebacObject;
};
```

to the HTTP call and converts the response back into `CheckResult`.

That conversion is the adapter pattern in its simplest form.

## Writing tuples

The adapter also implements `writeTuples`:

```ts
async writeTuples(tuples: TupleKey[]): Promise<void>
```

The documents service calls this at document-creation time to write the
workspace relationship for the new document — without leaking HTTP shapes
into the domain.

## The in-process stub (for tests)

Open `test/fixtures.ts`.

```ts
export const makeInProcessAuthzClient = (seed: TupleKey[] = []): AuthzClient => {
    const repository = makeInMemoryTupleRepository(seed);
    const evaluator  = makeGraphEvaluator({ repository });
    return {
        check:       req  => evaluator.evaluate(req),
        writeTuples: async tpls => { for (const t of tpls) repository.write(t); },
    };
};
```

This runs the real graph evaluator in-process. Tests do not need a running AuthZ
service. The same graph traversal logic is exercised without a network hop.

This is also why the `Evaluator` interface uses `Promise<CheckResult>`:

```ts
export interface Evaluator {
    evaluate: (request: CheckRequest) => Promise<CheckResult>;
}
```

The in-process call resolves immediately, but the interface is async to allow
real-world implementations (the AuthZ service over HTTP) to use the same port.

## Running services locally

Start both services:

```bash
npm run dev
```

Or separately:

```bash
npm run authz       # port 4100
npm run documents   # port 4000
```

The documents service reads `AUTHZ_URL` (default: `http://127.0.0.1:4100`) and
calls `makeAuthzServiceClient({ baseUrl: authzUrl })`.

Local runtime architecture:

```text
┌──────────────┐       HTTP        ┌──────────────────┐
│ terminal     │ ────────────────► │ documents :4000  │
│ client       │                   └────────┬─────────┘
└──────────────┘                            │ AuthzClient.check (HTTP)
                                            ▼
                                   ┌──────────────────┐
                                   │ authz    :4100   │
                                   │ GraphEvaluator   │
                                   └──────────────────┘
```

The documents service always talks to the AuthZ service over HTTP in production.
In tests, `makeInProcessAuthzClient` short-circuits the network entirely.

## TypeScript lesson

An HTTP adapter is just an external dependency. External dependencies should
have a small contact surface with your domain code.

That gives you:

- fast tests (no HTTP in unit tests)
- less churn when the API shape changes
- clearer business logic
- one place for HTTP error handling

This is a general TypeScript backend habit, not only an AuthZ habit.

## Exercise

Add a `deleteTuples` method to `makeAuthzServiceClient`.

Guidelines:

1. add `deleteTuples: (tuples: TupleKey[]) => Promise<void>` to the `AuthzClient` interface
2. implement the HTTP call inside `makeAuthzServiceClient`
3. update `makeInProcessAuthzClient` in `test/fixtures.ts` to implement the new method
4. do not expose HTTP-specific types in the document domain

## Checkpoint

Why does the document domain depend on `AuthzClient` instead of calling `fetch`
directly?

Good answer: the domain owns business rules; the adapter owns HTTP details. When
the AuthZ service URL or API shape changes, only the adapter changes.
