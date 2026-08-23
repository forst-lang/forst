import { describe, expect, test } from "bun:test";
import {
  CLOSED_METHODS,
  isClosedMethod,
  isStubMethod,
  STUB_METHODS,
} from "../../src/policy/methods.js";
import {
  METHOD_CALL,
  METHOD_GEN_RETURN,
  METHOD_INITIALIZE,
  METHOD_PING,
} from "../../src/rpc/protocol.js";

describe("policy method sets", () => {
  test("closed methods include initialize and ping", () => {
    expect(CLOSED_METHODS.has(METHOD_INITIALIZE)).toBe(true);
    expect(CLOSED_METHODS.has(METHOD_PING)).toBe(true);
    expect(CLOSED_METHODS.has(METHOD_CALL)).toBe(true);
  });

  test("isClosedMethod rejects unknown methods", () => {
    expect(isClosedMethod(METHOD_PING)).toBe(true);
    expect(isClosedMethod("forst.node/eval")).toBe(false);
  });

  test("stub methods are closed but flagged separately", () => {
    expect(STUB_METHODS.has(METHOD_GEN_RETURN)).toBe(true);
    expect(isClosedMethod(METHOD_GEN_RETURN)).toBe(true);
    expect(isStubMethod(METHOD_GEN_RETURN)).toBe(true);
    expect(isStubMethod(METHOD_PING)).toBe(false);
  });
});
