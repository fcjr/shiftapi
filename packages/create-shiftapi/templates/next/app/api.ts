import createClient from "openapi-react-query";
import { client, sse } from "@shiftapi/client";

export const api = createClient(client);
export { client, sse };
