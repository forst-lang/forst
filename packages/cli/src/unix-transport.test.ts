import { describe, expect, test } from "bun:test";
import { requestOverUnixSocket } from "./unix-transport.js";

describe("requestOverUnixSocket", () => {
  test("aborts immediately when signal is already aborted", async () => {
    if (process.platform === "win32") {
      return;
    }
    const controller = new AbortController();
    controller.abort();
    await expect(
      requestOverUnixSocket("/tmp/forst-unix-abort-test.sock", "/health", {
        signal: controller.signal,
      })
    ).rejects.toMatchObject({ name: "AbortError" });
  });
});
