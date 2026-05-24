import makeServerApp from "./compose.ts";

const app = makeServerApp();

await app.documents.create(app.seedDocument);

app.server.listen(app.port, () => {
    console.log(`TS ReBAC server listening on http://127.0.0.1:${app.port}`);
});
