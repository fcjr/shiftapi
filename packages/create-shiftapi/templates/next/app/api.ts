import createClient from "openapi-react-query";
import { client } from "@shiftapi/client";

export const api = createClient(client);
export { client };
