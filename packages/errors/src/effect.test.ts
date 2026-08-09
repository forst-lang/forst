import { describe, expect, test } from "bun:test";
import { Equal } from "effect";
import { InvokeRejected, isInvokeFailure } from "./effect.js";

describe("@forst/errors/effect", () => {
  test("InvokeRejected uses Data.TaggedError with namespaced tag", () => {
    const err = new InvokeRejected({
      packageName: "main",
      functionName: "Echo",
    });
    expect(err._tag).toBe("@forst/errors/InvokeRejected");
    expect(isInvokeFailure(err)).toBe(true);
    expect(Equal.equals(err, new InvokeRejected(err))).toBe(true);
  });
});
