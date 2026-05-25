// AuthZ service entrypoint.
//
// Starts the authorization engine that stores relationship tuples and
// evaluates the ReBAC permission graph.
//
// Ports:
//   POST /check     — check a permission
//   POST /tuples    — write relationship tuples
//   DELETE /tuples  — remove relationship tuples
//   GET  /tuples    — list tuples (audit)
//
// Port: AUTHZ_PORT (default 4100)

import composeAuthzService from "./compose.ts";
import { seedPolicyTuples } from "../../test/fixtures.ts";

const { listen } = composeAuthzService({ seedTuples: seedPolicyTuples() });

listen(port => {
    console.log(`AuthZ service → http://127.0.0.1:${port}`);
    console.log(`  POST   /check`);
    console.log(`  POST   /tuples`);
    console.log(`  DELETE /tuples`);
    console.log(`  GET    /tuples`);
});
