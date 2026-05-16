import { createDemoApp } from "./app/create-demo.js";

const app = createDemoApp();

for (const actor of app.actors) {
  const result = await app.authorizer.check({
    user: actor,
    relation: "can_edit",
    object: app.document
  });

  console.log(`${actor} can_edit ${app.document}: ${result.allowed}`);
  console.log(result.trace.map((line) => `  ${line}`).join("\n"));
}
