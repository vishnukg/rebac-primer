# ADR 0001 — Composition roots return only what the entry point drives

**Status:** Accepted (2026-06-07)

## Context

This codebase wires each service in a **composition root** (`compose.ts`) that an
**entry point** (`index.ts`) starts. The root builds the adapters and domain and
hands back the capability the entry point runs:

```ts
const { listen } = composeAuthzService({ seedTuples: seedPolicyTuples() });
listen(/* … */);
```

Over time the roots drifted into returning **more than the entry point uses**:

- `composeAuthzService` returned `{ listen, domain }`, but `index.ts` used only
  `listen`, and no test referenced `domain` — it was **dead**.
- `composeDocumentsService` returned `{ listen, documents }` purely so `index.ts`
  could seed a demo document at startup via `documents.create(...)`.

Returning the domain (`domain` / `documents`) from a composition root **re-exposes
the very internals the root exists to encapsulate**. It also made the two
services inconsistent: authz took its seed as config (`seedTuples`) and returned
`{ listen }`, while documents seeded *outside* the root and handed its domain back
out to do it.

(The same smell appeared in the ModulePattern reference repo, where
`composeServerApp` returned `{ listen, restaurant }` only so an integration test
could reach the domain.)

## Decision

**A composition root returns only the capability (or capabilities) its entry
point actually drives.** Startup data is passed *in* as config, and the root
performs any seeding *internally*:

```ts
composeAuthzService({ seedTuples })       // → { listen }
composeDocumentsService({ seedDocuments }) // → { listen }
composeCliApp()                            // → { run }
```

The domain is never returned for a side task (seeding) or for tests.

## Why the documents service needed special handling

Seeding is not the same operation in both services, and that difference is the
whole reason `documents` had leaked out:

| | Authz tuples | A document |
| --- | --- | --- |
| What it is | static relationship data | a full domain operation |
| Path | `makeInMemoryTupleRepository({ seed })` | `create → authz.check → repo.save → authz.writeTuples` |
| When it can run | at **construction** (synchronous, no deps) | only **after startup** — needs the HTTP server up and the authz service reachable |

So a document seed cannot be a constructor argument the way authz tuples are. The
fix is to still take it as config (`seedDocuments`) but run it **inside
`listen`**, after the socket is bound and before `onReady` is signalled:

```ts
const listen = (onReady) => {
  server.listen(port, "0.0.0.0", () => {
    void seed().then(() => onReady(port)); // seed runs here, post-startup
  });
};
return { listen };
```

The seed must use the **same** `documents` instance the server serves from (same
in-memory repository), which is exactly why the entry point couldn't just compose
its own domain to seed with — see Alternatives.

## Consequences

- Both service roots return `{ listen }`; the CLI root returns `{ run }`. The
  composition surface is minimal and the domain stays encapsulated.
- Demo seed data lives in `src/demo/fixtures.ts` (`seedPolicyTuples`,
  `seedDocuments`), consumed by the entry points and re-used by tests — so the
  demo and the tests stay in sync.
- Tests/integration that need the domain build it **directly** via the shared
  domain factory (`makeDocuments`) or the relevant `make*` factories — they
  never reach into a service root. This is what made the exposed domain
  redundant in the first place.
- Trade-off accepted: `listen()` now performs a startup domain op (seeding).
  That's slightly more than pure wiring, but startup orchestration already lives
  in `listen` (it binds the socket and handles errors), so seeding-on-start fits.

## Alternatives considered

1. **Keep `{ listen, documents }` and seed in `index.ts`.** Rejected: re-exposes
   the domain and is asymmetric with authz. The same "return only what's driven"
   rule we apply everywhere else would flag it.
2. **Let `index.ts` build its own `documents` (via `makeDocuments`) to
   seed.** Rejected: that builds a *second* domain over a *different* in-memory
   repository, so the seed would never reach the store the server actually
   serves. The seed must use the same instance the server uses.
3. **Seed over HTTP after startup.** Rejected: indirect, needs the server to call
   itself, and duplicates request plumbing.

## Related

- `docs/19-factory-function-pattern.md` — the make/compose rule and the
  "return only what's driven" principle.
- ModulePattern reference repo: `composeServerApp` was trimmed to `{ listen }`;
  its integration tests call `composeRestaurant` directly.
