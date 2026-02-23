import createClient from "openapi-svelte-query";
import { client } from "@shiftapi/client";

export const $api = createClient(client);
