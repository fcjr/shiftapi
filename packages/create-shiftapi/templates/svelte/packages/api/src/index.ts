import createClient from "shiftapi/client/svelte";
import { client, sse } from "@shiftapi/client";

export const api = createClient(client);
export { client, sse };
