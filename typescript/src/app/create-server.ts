import type { Server } from "node:http";
import { createHttpServer } from "../http/server.js";
import { createServices } from "./create-services.js";

export type ServerApp = Readonly<{
  port: number;
  server: Server;
}>;

export async function createServerApp(env: NodeJS.ProcessEnv = process.env): Promise<ServerApp> {
  const services = await createServices();

  return {
    port: readPort(env.PORT, 4000),
    server: createHttpServer({ documents: services.documents })
  };
}

function readPort(value: string | undefined, fallback: number): number {
  if (value === undefined || value.trim() === "") {
    return fallback;
  }

  const port = Number(value);
  if (!Number.isInteger(port) || port < 1 || port > 65_535) {
    throw new Error(`Invalid PORT: ${value}`);
  }

  return port;
}
