import createClient from "openapi-react-query";
import { client, subscribe } from "@shiftapi/client";

export const api = createClient(client);
export { client, subscribe };
