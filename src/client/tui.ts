import { createClientApp } from "../app/create-client.js";

const app = createClientApp();

try {
  await app.run();
} finally {
  app.terminal.close();
}
