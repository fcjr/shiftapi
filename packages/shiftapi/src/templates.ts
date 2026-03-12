function indent(text: string, spaces = 2): string {
  const prefix = " ".repeat(spaces);
  return text
    .split("\n")
    .map((line) => (line ? prefix + line : line))
    .join("\n");
}

/**
 * Inline JS for a bodySerializer that auto-detects File/Blob values
 * and wraps the body in FormData. Falls back to JSON.stringify otherwise.
 */
const BODY_SERIALIZER = `(body) => {
  if (typeof body !== "object" || body === null) return JSON.stringify(body);
  const isBinary = (v) => v instanceof Blob || v instanceof File || v instanceof Uint8Array;
  const values = Object.values(body);
  const hasFile = values.some(
    (v) => isBinary(v) || (Array.isArray(v) && v.some(isBinary)),
  );
  if (!hasFile) return JSON.stringify(body);
  const toBlob = (v) => v instanceof Uint8Array ? new Blob([v]) : v;
  const fd = new FormData();
  for (const [key, value] of Object.entries(body)) {
    if (value === undefined || value === null) continue;
    if (Array.isArray(value)) {
      for (const item of value) fd.append(key, isBinary(item) ? toBlob(item) : String(item));
    } else {
      fd.append(key, isBinary(value) ? toBlob(value) : String(value));
    }
  }
  return fd;
}`;

/**
 * Builds the WSChannels type declaration from an AsyncAPI spec.
 * Maps each channel to its send (subscribe) and receive (publish) schema
 * types, referencing the openapi-typescript-generated component schemas.
 */
function buildWSChannelsType(asyncapiSpec: object | null): string {
  if (!asyncapiSpec) return "";

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const spec = asyncapiSpec as any;
  const channels = spec.channels;
  if (!channels || Object.keys(channels).length === 0) return "";

  const entries: string[] = [];
  let hasErrors = false;
  for (const [path, channel] of Object.entries(channels) as [string, any][]) {
    const sendType = resolveMessageType(channel.subscribe?.message, spec);
    const recvType = resolveMessageType(channel.publish?.message, spec);
    const xErrors = channel["x-errors"];
    let errorsType = "Record<number, unknown>";
    if (xErrors && typeof xErrors === "object") {
      hasErrors = true;
      const errEntries = Object.entries(xErrors)
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        .map(([code, schema]: [string, any]) => {
          const schemaType = resolvePayloadType(schema, spec);
          return `${code}: ${schemaType}`;
        })
        .filter((e) => !e.endsWith("unknown"));
      if (errEntries.length > 0) {
        errorsType = `{ ${errEntries.join("; ")} }`;
      }
    }
    entries.push(`    ${JSON.stringify(path)}: {\n      send: ${sendType};\n      receive: ${recvType};\n      errors: ${errorsType};\n    };`);
  }

  const wsErrorType = hasErrors
    ? `\n  type WSErrorFor<P extends WSPaths> = { [C in keyof WSChannels[P]["errors"]]: WSError<C, WSChannels[P]["errors"][C]> }[keyof WSChannels[P]["errors"]];`
    : "";

  return `
  interface WSChannels {
${entries.join("\n")}
  }

  type WSPaths = keyof WSChannels;
  type WSSend<P extends WSPaths> = WSChannels[P]["send"];
  type WSRecv<P extends WSPaths> = WSChannels[P]["receive"];${wsErrorType}

  export function websocket<P extends WSPaths>(
    path: P,
    options?: { params?: Record<string, unknown>; protocols?: string[] }
  ): WSConnection<WSSend<P>, WSRecv<P>>;
`;
}

/**
 * Resolves an AsyncAPI message definition to a TypeScript type string.
 * Handles inline messages (single-type channels) and oneOf
 * (discriminated union variants with {type, data} envelopes).
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function resolveMessageType(message: any, spec: any): string {
  if (!message) return "unknown";

  // Inline message with payload (single-message channels)
  if (message.payload) {
    return resolvePayloadType(message.payload, spec);
  }

  // oneOf — discriminated union
  if (message.oneOf) {
    const variants = message.oneOf
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      .map((m: any) => {
        if (m.$ref) {
          const msgName = m.$ref.replace("#/components/messages/", "");
          const msg = spec.components?.messages?.[msgName];
          if (!msg) return "unknown";
          return resolveEnvelopeType(msg.payload, spec);
        }
        return resolveEnvelopeType(m.payload, spec);
      })
      .filter((t: string) => t !== "unknown");
    return variants.length > 0 ? variants.join(" | ") : "unknown";
  }

  return "unknown";
}

/**
 * Resolves a payload schema to a TypeScript type referencing
 * openapi-typescript-generated component schemas.
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function resolvePayloadType(payload: any, _spec: any): string {
  if (!payload) return "unknown";
  if (payload.$ref) {
    const name = payload.$ref.replace("#/components/schemas/", "");
    return `components["schemas"]["${name}"]`;
  }
  return "unknown";
}

/**
 * Resolves an envelope schema ({type, data}) to a TypeScript
 * discriminated union variant type.
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function resolveEnvelopeType(payload: any, spec: any): string {
  if (!payload) return "unknown";

  // If it's a $ref to an envelope schema in components
  if (payload.$ref) {
    const name = payload.$ref.replace("#/components/schemas/", "");
    const schema = spec.components?.schemas?.[name];
    if (!schema) return "unknown";
    return resolveEnvelopeFromSchema(schema, spec);
  }

  return resolveEnvelopeFromSchema(payload, spec);
}

/**
 * Given an inline envelope schema with {type: enum, data: $ref},
 * produces a TypeScript type like { type: "chat"; data: ChatMessage }.
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function resolveEnvelopeFromSchema(schema: any, spec: any): string {
  const typeEnum = schema.properties?.type?.enum;
  const dataRef = schema.properties?.data;
  if (!typeEnum || typeEnum.length !== 1 || !dataRef) return "unknown";
  const typeName = JSON.stringify(typeEnum[0]);
  const dataType = resolvePayloadType(dataRef, spec);
  return `{ type: ${typeName}; data: ${dataType} }`;
}

/**
 * Extracts schema names from the openapi-typescript generated types string
 * and produces `export type Name = components["schemas"]["Name"];` aliases.
 */
function buildSchemaExports(generatedTypes: string): string {
  const schemaNames: string[] = [];
  // Match the schemas block inside `components { schemas: { ... } }`
  const schemasMatch = generatedTypes.match(/schemas:\s*\{([\s\S]*?)\n {4}\}/);
  if (schemasMatch) {
    // Each top-level schema key is at 8-space indent
    const re = /^ {8}(\w+)\s*:/gm;
    let m;
    while ((m = re.exec(schemasMatch[1])) !== null) {
      schemaNames.push(m[1]);
    }
  }
  if (schemaNames.length === 0) return "";
  return schemaNames
    .map((name) => `  export type ${name} = components["schemas"]["${name}"];`)
    .join("\n");
}

export function dtsTemplate(generatedTypes: string, asyncapiSpec: object | null = null): string {
  const wsSection = buildWSChannelsType(asyncapiSpec);
  const wsImport = wsSection ? '\n  import type { WSConnection } from "shiftapi/internal/browser";\n  import { WSError } from "shiftapi/internal/browser";' : "";
  const schemaExports = buildSchemaExports(generatedTypes);

  return `\
// Auto-generated by shiftapi. Do not edit.
declare module "@shiftapi/client" {
${indent(generatedTypes)}

  import type createClient from "openapi-fetch";
  import type { SSEStream } from "shiftapi/internal/browser";${wsImport}

  type SSEPaths = {
    [P in keyof paths]: paths[P] extends {
      get: { responses: { 200: { content: { "text/event-stream": infer T } } } }
    } ? P : never
  }[keyof paths];

  type SSEEventType<P extends SSEPaths> =
    paths[P] extends {
      get: { responses: { 200: { content: { "text/event-stream": infer T } } } }
    } ? T : never;

  type SSEParams<P extends SSEPaths> =
    paths[P] extends { get: { parameters: infer Params } } ? Params : never;

  export function sse<P extends SSEPaths>(
    path: P,
    options?: { params?: SSEParams<P>; signal?: AbortSignal }
  ): SSEStream<SSEEventType<P>>;
${wsSection}
  export const client: ReturnType<typeof createClient<paths>>;
  export { createClient };${wsSection ? "\n  export { WSError };" : ""}
${schemaExports ? "\n" + schemaExports + "\n" : ""}}
`;
}

export function clientJsTemplate(baseUrl: string, options?: { hasWebSocket?: boolean }): string {
  const hasWS = options?.hasWebSocket ?? false;
  const wsImport = hasWS ? ", WSError" : "";
  const wsExport = hasWS ? ", WSError" : "";
  return `\
// Auto-generated by shiftapi. Do not edit.
import createClient from "openapi-fetch";
import { createSSE, createWebSocket${wsImport} } from "shiftapi/internal/browser";

/** Pre-configured, fully-typed API client. */
export const client = createClient({
  baseUrl: ${JSON.stringify(baseUrl)},
  bodySerializer: ${BODY_SERIALIZER},
});

export const sse = createSSE(${JSON.stringify(baseUrl)});
export const websocket = createWebSocket(${JSON.stringify(baseUrl)});

export { createClient${wsExport} };
`;
}

export function nextClientJsTemplate(
  port: number,
  baseUrl: string,
  devApiPrefix?: string,
  options?: { hasWebSocket?: boolean },
): string {
  const hasWS = options?.hasWebSocket ?? false;
  const wsImport = hasWS ? ", WSError" : "";
  const wsExport = hasWS ? ", WSError" : "";
  if (!devApiPrefix) {
    return `\
// Auto-generated by @shiftapi/next. Do not edit.
import createClient from "./openapi-fetch.js";
import { createSSE, createWebSocket${wsImport} } from "shiftapi/internal/browser";

const baseUrl =
  process.env.NEXT_PUBLIC_SHIFTAPI_BASE_URL || ${JSON.stringify(baseUrl)};

/** Pre-configured, fully-typed API client. */
export const client = createClient({
  baseUrl,
  bodySerializer: ${BODY_SERIALIZER},
});

export const sse = createSSE(baseUrl);
export const websocket = createWebSocket(baseUrl);

export { createClient${wsExport} };
`;
  }

  const devServerUrl = `http://localhost:${port}`;
  return `\
// Auto-generated by @shiftapi/next. Do not edit.
import createClient from "./openapi-fetch.js";
import { createSSE, createWebSocket${wsImport} } from "shiftapi/internal/browser";

const baseUrl =
  process.env.NEXT_PUBLIC_SHIFTAPI_BASE_URL ||
  (typeof window !== "undefined"
    ? ${JSON.stringify(devApiPrefix)}
    : ${JSON.stringify(devServerUrl)});

/** Pre-configured, fully-typed API client. */
export const client = createClient({
  baseUrl,
  bodySerializer: ${BODY_SERIALIZER},
});

export const sse = createSSE(baseUrl);
export const websocket = createWebSocket(baseUrl);

export { createClient${wsExport} };
`;
}

export function virtualModuleTemplate(
  baseUrl: string,
  devApiPrefix?: string,
  options?: { hasWebSocket?: boolean },
): string {
  const hasWS = options?.hasWebSocket ?? false;
  const wsImport = hasWS ? ", WSError" : "";
  const wsExport = hasWS ? ", WSError" : "";
  const baseUrlExpr = devApiPrefix
    ? `import.meta.env.VITE_SHIFTAPI_BASE_URL || (import.meta.env.DEV ? ${JSON.stringify(devApiPrefix)} : ${JSON.stringify(baseUrl)})`
    : `import.meta.env.VITE_SHIFTAPI_BASE_URL || ${JSON.stringify(baseUrl)}`;

  return `\
// Auto-generated by @shiftapi/vite-plugin
import createClient from "openapi-fetch";
import { createSSE, createWebSocket${wsImport} } from "shiftapi/internal/browser";

const baseUrl = ${baseUrlExpr};

/** Pre-configured, fully-typed API client. */
export const client = createClient({
  baseUrl,
  bodySerializer: ${BODY_SERIALIZER},
});

export const sse = createSSE(baseUrl);
export const websocket = createWebSocket(baseUrl);

export { createClient${wsExport} };
`;
}
