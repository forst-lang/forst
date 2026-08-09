import { describe, expect, test } from "bun:test";
import {
  ContractVersionMismatch,
  ForstTestServerFailed,
  ForstUnknownFailure,
  InvokeRejected,
  isInvokeFailure,
} from "./index.js";

describe("@forst/errors (Promise mode)", () => {
  test("InvokeRejected has flat _tag and instanceof works", () => {
    const err = new InvokeRejected({
      packageName: "main",
      functionName: "Echo",
      serverError: "nope",
    });
    expect(err._tag).toBe("@forst/errors/InvokeRejected");
    expect(err instanceof InvokeRejected).toBe(true);
    expect(isInvokeFailure(err)).toBe(true);
  });

  test("isInvokeFailure rejects domain-style errors", () => {
    const err = new ForstUnknownFailure({ message: "unknown" });
    expect(isInvokeFailure(err)).toBe(false);
  });

  test("harness and unknown failure classes export flat tags", () => {
    const harness = new ForstTestServerFailed({ reason: "spawn_failed" });
    const unknown = new ForstUnknownFailure({ message: "x" });
    expect(harness._tag).toBe("@forst/errors/ForstTestServerFailed");
    expect(unknown._tag).toBe("@forst/errors/ForstUnknownFailure");
  });

  test("ContractVersionMismatch is not an invoke failure guard match without tag", () => {
    const err = new ContractVersionMismatch({
      expectedContractVersion: "1",
      serverContractVersion: "2",
    });
    expect(isInvokeFailure(err)).toBe(true);
  });
});
