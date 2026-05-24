import makeCliApp from "./compose.ts";

const app = makeCliApp();

try {
  await app.run();
} finally {
  app.terminal.close();
}
