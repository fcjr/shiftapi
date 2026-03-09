import createClient from "openapi-svelte-query";
import { client, subscribe } from "@shiftapi/client";

export const api = createClient(client);
export { client, subscribe };
