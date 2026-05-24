# TypeScript ReBAC Primer

This implementation uses the same module pattern as the reference project:

```text
src/modules/<module>/index.ts  public API
src/modules/<module>/make*.ts  factories
src/server/compose.ts          server wiring
src/cli/compose.ts             terminal-client wiring
src/demo/compose.ts            demo wiring
```

The learning flow is:

1. Authn: `modules/authn` verifies a demo OAuth2-style bearer token and returns `user:*`.
2. Authz: `modules/authz` answers ReBAC checks from relationship tuples.
3. Documents: `modules/documents` protects create/read/update with ReBAC.
4. HTTP: `modules/http` turns requests into authn/authz-aware document calls.
5. Composition: `server/compose.ts` wires concrete adapters together.

Useful commands:

```bash
npm install
npm run check
npm run demo
npm run server
npm run client
```

Demo tokens for HTTP:

```text
demo-token-alice  user:alice, can read and edit
demo-token-bob    user:bob, can read only
demo-token-casey  user:casey, authenticated but denied by ReBAC
```
