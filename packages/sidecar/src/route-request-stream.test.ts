import { describe, expect, it } from "bun:test";
import { Readable } from "node:stream";
import { text } from "node:stream/consumers";

import {
  createInvokeRawJsonBodyReadable,
  hasMaterializedBody,
  splitRawInvokeJsonArray,
} from "./route-request-stream";

describe("splitRawInvokeJsonArray", () => {
  it("splits JSON around bodyBase64 for streaming insertion", () => {
    const { prefix, suffix } = splitRawInvokeJsonArray({
      method: "POST",
      url: "/a",
      path: "/a",
      query: {},
      headers: {},
    });
    expect(prefix.endsWith('"bodyBase64":"')).toBe(true);
    expect(suffix).toBe('"}]');
    expect(prefix.startsWith("[{")).toBe(true);
  });
});

describe("createInvokeRawJsonBodyReadable", () => {
  it("emits valid JSON matching buffered invoke for same payload", async () => {
    const payload = Buffer.from("hello");
    const req = Readable.from([payload]) as any;
    req.method = "POST";
    req.originalUrl = "/z";
    req.url = "/z";
    req.path = "/z";
    req.query = {};
    req.headers = {};

    const r = createInvokeRawJsonBodyReadable(req, {
      method: "POST",
      url: "/z",
      path: "/z",
      query: {},
      headers: {},
    });
    const json = await text(r);
    const parsed = JSON.parse(json) as Array<{
      bodyBase64: string;
    }>;
    expect(parsed).toHaveLength(1);
    expect(parsed[0].bodyBase64).toBe(payload.toString("base64"));
  });
});

describe("hasMaterializedBody", () => {
  it("is true when req.body is set", () => {
    expect(
      hasMaterializedBody({ body: Buffer.from("x") } as any)
    ).toBe(true);
  });

  it("is false for a bare readable stream", () => {
    const req = Readable.from([]) as any;
    expect(hasMaterializedBody(req)).toBe(false);
  });
});
