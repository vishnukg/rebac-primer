import { createServices } from "./app/create-services.js";
import { createHttpServer } from "./http/server.js";

const port = Number(process.env.PORT ?? "4000");
const services = await createServices();
const server = createHttpServer({ documents: services.documents });

server.listen(port, () => {
  console.log(`TS ReBAC server listening on http://127.0.0.1:${port}`);
});
