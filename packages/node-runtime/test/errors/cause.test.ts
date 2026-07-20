import { describe, expect, test } from "bun:test";
import { causeToError, UnknownCauseError } from "../../src/errors/cause.js";

describe("causeToError", () => {
  test("returns Error instances unchanged", () => {
    const err = new Error("boom");
    expect(causeToError(err)).toBe(err);
  });

  test("wraps non-Error values in UnknownCauseError", () => {
    const wrapped = causeToError("plain string");
    expect(wrapped).toBeInstanceOf(UnknownCauseError);
    expect(wrapped.message).toBe("plain string");
  });
});
