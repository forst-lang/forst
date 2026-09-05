import { describe, expect, test } from "bun:test";
import { resolveExportValue } from "../../src/runtime/export_value.js";

describe("resolveExportValue", () => {
  test("returns named export when present", () => {
    const mod = { add: (a: number, b: number) => a + b };
    expect(resolveExportValue(mod, "add")).toBe(mod.add);
  });

  test("reads export from default object", () => {
    const fn = () => "ok";
    const mod = { default: { greet: fn } };
    expect(resolveExportValue(mod, "greet")).toBe(fn);
  });

  test("returns undefined when export is missing", () => {
    expect(resolveExportValue({ default: {} }, "missing")).toBeUndefined();
    expect(resolveExportValue({}, "missing")).toBeUndefined();
  });
});
