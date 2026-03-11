import { describe, it, expect } from "vitest";
import { dtsTemplate } from "../templates";

describe("dtsTemplate", () => {
  it("generates WSChannels with x-errors", () => {
    const asyncapiSpec = {
      asyncapi: "2.4.0",
      channels: {
        "/ws": {
          subscribe: {
            message: {
              payload: { $ref: "#/components/schemas/ServerMsg" },
            },
          },
          publish: {
            message: {
              payload: { $ref: "#/components/schemas/ClientMsg" },
            },
          },
          "x-errors": {
            "4401": { $ref: "#/components/schemas/AuthError" },
            "4422": { $ref: "#/components/schemas/ValidationError" },
          },
        },
      },
      components: {
        schemas: {
          ServerMsg: { type: "object", properties: { text: { type: "string" } } },
          ClientMsg: { type: "object", properties: { text: { type: "string" } } },
          AuthError: { type: "object", properties: { message: { type: "string" } } },
          ValidationError: { type: "object", properties: { message: { type: "string" } } },
        },
      },
    };

    const output = dtsTemplate("// types", asyncapiSpec);

    // Should have errors in the WSChannels interface.
    expect(output).toContain("errors:");
    expect(output).toContain('4401: components["schemas"]["AuthError"]');
    expect(output).toContain('4422: components["schemas"]["ValidationError"]');

    // Should generate WSErrorFor type helper.
    expect(output).toContain("WSErrorFor");
  });

  it("generates WSChannels without errors when no x-errors", () => {
    const asyncapiSpec = {
      asyncapi: "2.4.0",
      channels: {
        "/ws": {
          subscribe: {
            message: {
              payload: { $ref: "#/components/schemas/ServerMsg" },
            },
          },
          publish: {
            message: {
              payload: { $ref: "#/components/schemas/ClientMsg" },
            },
          },
        },
      },
      components: {
        schemas: {
          ServerMsg: { type: "object", properties: { text: { type: "string" } } },
          ClientMsg: { type: "object", properties: { text: { type: "string" } } },
        },
      },
    };

    const output = dtsTemplate("// types", asyncapiSpec);

    // Should still have errors field with default type.
    expect(output).toContain("errors: Record<number, unknown>");
    // Should not generate WSErrorFor when no channels have x-errors.
    expect(output).not.toContain("WSErrorFor");
  });

  it("exports standalone type aliases for component schemas", () => {
    // Simulate openapi-typescript output with 4-space indentation
    const generatedTypes = `export interface components {
    schemas: {
        EchoReply: {
            text: string;
        };
        ChatMsg: {
            user: string;
            text: string;
        };
    };
}`;

    const output = dtsTemplate(generatedTypes);

    expect(output).toContain('export type EchoReply = components["schemas"]["EchoReply"];');
    expect(output).toContain('export type ChatMsg = components["schemas"]["ChatMsg"];');
  });

  it("handles generated types with no schemas gracefully", () => {
    const output = dtsTemplate("// no schemas here");
    expect(output).not.toContain("export type ");
    // Should still produce a valid module declaration
    expect(output).toContain('declare module "@shiftapi/client"');
  });
});
