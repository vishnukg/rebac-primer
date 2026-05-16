import { createServerApp } from "./app/create-server.js";

const app = await createServerApp();

app.server.listen(app.port, () => {
  console.log(`TS ReBAC server listening on http://127.0.0.1:${app.port}`);
});
