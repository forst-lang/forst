import { describe, expect, it } from "bun:test";
import {
  DEFAULT_SANITIZE_OPTIONS,
  invokeRequestLogFields,
  invokeResponseLogFields,
  sanitizePayload,
  sanitizeRequestBodyString,
} from "./sanitizeLogPayload";

describe("sanitizePayload", () => {
  it("passes through null and undefined", () => {
    expect(sanitizePayload(null)).toBeNull();
    expect(sanitizePayload(undefined)).toBeUndefined();
  });

  it("truncates long strings", () => {
    const long = "x".repeat(300);
    const out = sanitizePayload(long, 0, {
      ...DEFAULT_SANITIZE_OPTIONS,
      maxStringLen: 10,
    });
    expect(typeof out).toBe("string");
    expect((out as string).length).toBeLessThan(long.length);
    expect((out as string)).toContain("…");
  });

  it("returns [max depth] beyond maxDepth", () => {
    const nested = { a: { b: { c: 1 } } };
    expect(
      sanitizePayload(nested, 0, { ...DEFAULT_SANITIZE_OPTIONS, maxDepth: 0 })
    ).toEqual({ a: "[max depth]" });
    expect(
      sanitizePayload(nested, 0, { ...DEFAULT_SANITIZE_OPTIONS, maxDepth: 1 })
    ).toEqual({ a: { b: "[max depth]" } });
  });

  it("handles primitives", () => {
    expect(sanitizePayload(42)).toBe(42);
    expect(sanitizePayload(true)).toBe(true);
    expect(sanitizePayload(1n)).toBe("1n");
    expect(sanitizePayload(() => {})).toBe("[Function]");
    expect(sanitizePayload(Symbol("s"))).toBe("Symbol(s)");
  });

  it("redacts sensitive object keys", () => {
    expect(
      sanitizePayload({ password: "secret", ok: 1 })
    ).toEqual({ password: "[redacted]", ok: 1 });
  });

  it("truncates large arrays", () => {
    const arr = Array.from({ length: 30 }, (_, i) => i);
    const out = sanitizePayload(arr, 0, {
      ...DEFAULT_SANITIZE_OPTIONS,
      maxArrayItems: 3,
    }) as unknown[];
    expect(out).toHaveLength(4);
    expect(out[3]).toMatch(/more items/);
  });

  it("caps object keys", () => {
    const obj: Record<string, number> = {};
    for (let i = 0; i < 40; i++) obj[`k${i}`] = i;
    const out = sanitizePayload(obj, 0, {
      ...DEFAULT_SANITIZE_OPTIONS,
      maxObjectKeys: 2,
    }) as Record<string, unknown>;
    expect(Object.keys(out).length).toBeGreaterThan(2);
    expect(out["…"]).toMatch(/more keys/);
  });

  it("sanitizes Headers-like objects", () => {
    const h = new Headers();
    h.set("X-Ok", "yes");
    h.set("Authorization", "bearer token");
    const out = sanitizePayload(h) as Record<string, string>;
    const authKey = Object.keys(out).find(
      (k) => k.toLowerCase() === "authorization"
    );
    expect(authKey).toBeDefined();
    expect(out[authKey!]).toBe("[redacted]");
    const keys = Object.keys(out);
    expect(keys.some((k) => k.toLowerCase() === "x-ok")).toBe(true);
  });

  it("sanitizes nested headers property case-insensitively", () => {
    const out = sanitizePayload({
      headers: { Authorization: "x", safe: "y" },
    }) as { headers: Record<string, unknown> };
    expect(out.headers.Authorization).toBe("[redacted]");
    expect(out.headers.safe).toBe("y");
  });
});

describe("sanitizeRequestBodyString", () => {
  it("returns undefined for undefined input", () => {
    expect(sanitizeRequestBodyString(undefined)).toBeUndefined();
  });

  it("returns byte note when body exceeds parse limit", () => {
    const huge = "x".repeat(70 * 1024);
    expect(sanitizeRequestBodyString(huge)).toEqual({
      bytes: huge.length,
      note: "body too large for log preview",
    });
  });

  it("parses JSON and sanitizes", () => {
    expect(sanitizeRequestBodyString('{"password":"p","n":1}')).toEqual({
      password: "[redacted]",
      n: 1,
    });
  });

  it("returns preview for non-JSON small bodies", () => {
    const out = sanitizeRequestBodyString("not json {{{") as {
      bytes: number;
      preview: string;
    };
    expect(out.bytes).toBe(12);
    expect(out.preview).toContain("not json");
  });
});

describe("invokeResponseLogFields", () => {
  it("truncates error and sanitizes result", () => {
    const err = "e".repeat(300);
    const fields = invokeResponseLogFields({
      success: false,
      error: err,
      result: { token: "t" },
    });
    expect(fields.success).toBe(false);
    expect(String(fields.error).length).toBeLessThan(err.length);
    expect((fields.result as { token: string }).token).toBe("[redacted]");
  });
});

describe("invokeRequestLogFields", () => {
  it("includes arg metadata and sanitizes args", () => {
    const fields = invokeRequestLogFields({
      package: "p",
      function: "f",
      args: { secret: "x", n: 1 },
      streaming: true,
    });
    expect(fields.package).toBe("p");
    expect(fields.function).toBe("f");
    expect(fields.streaming).toBe(true);
    expect(fields.argCount).toBe(2);
    expect(typeof fields.argsPayloadBytes).toBe("number");
    expect((fields.args as { secret: string }).secret).toBe("[redacted]");
  });

  it("uses argCount -1 when args are not JSON-serializable", () => {
    const circular: Record<string, unknown> = {};
    circular.self = circular;
    const fields = invokeRequestLogFields({
      package: "p",
      function: "f",
      args: circular,
    });
    expect(fields.argCount).toBe(-1);
    expect(fields.argsPayloadBytes).toBe(0);
  });
});
