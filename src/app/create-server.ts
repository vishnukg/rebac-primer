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
    port: Number(env.PORT ?? "4000"),
    server: createHttpServer({ documents: services.documents })
  };
}
