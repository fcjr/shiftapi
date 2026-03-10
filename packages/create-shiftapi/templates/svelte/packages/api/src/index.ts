import createClient from "openapi-svelte-query";
import { client, sse } from "@shiftapi/client";

export const api = createClient(client);
export { client, sse };
