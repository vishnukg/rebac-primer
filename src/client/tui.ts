import { createInterface } from "node:readline/promises";
import { stdin as input, stdout as output } from "node:process";
import { RebacApiClient } from "./api-client.js";

const client = new RebacApiClient(process.env.REBAC_API_URL ?? "http://127.0.0.1:4000");
const terminal = createInterface({ input, output });

try {
  await run();
} finally {
  terminal.close();
}

async function run(): Promise<void> {
  console.log("TS ReBAC client");
  console.log("Actors: alice can edit, bob can read, chandra is denied by default.");

  const healthy = await client.health();
  if (!healthy) {
    throw new Error("Server health check failed");
  }

  let running = true;
  while (running) {
    console.log("\n1. Read roadmap");
    console.log("2. Update roadmap");
    console.log("3. Exit");

    const choice = await terminal.question("Choose: ");

    if (choice === "1") {
      await readRoadmap();
    } else if (choice === "2") {
      await updateRoadmap();
    } else if (choice === "3") {
      running = false;
    } else {
      console.log("Unknown choice");
    }
  }
}

async function readRoadmap(): Promise<void> {
  const actorId = await terminal.question("Actor id: ");
  const document = await client.readDocument("roadmap", actorId);
  console.log(`\n${document.title}`);
  console.log(document.body);
  console.log(`updated by ${document.updatedBy}`);
}

async function updateRoadmap(): Promise<void> {
  const actorId = await terminal.question("Actor id: ");
  const body = await terminal.question("New body: ");
  const document = await client.updateDocument("roadmap", actorId, body);
  console.log(`Updated ${document.id}; updated by ${document.updatedBy}`);
}
