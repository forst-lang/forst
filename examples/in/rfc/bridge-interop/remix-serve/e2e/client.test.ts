import { describe, expect, test } from "bun:test";
import { $main } from "@forst/gen/main";

describe("remix-serve generated client", () => {
  test("ListTodos and GetDashboard over embedded invoke", async () => {
    process.env.FORST_SKIP_SPAWN = "1";
    process.env.FORST_BASE_URL =
      process.env.FORST_BASE_URL ?? "http://127.0.0.1:6321";

    const list = await $main.ListTodos();
    expect(list.open).toBeGreaterThanOrEqual(0);

    const dashboard = await $main.GetDashboard();
    expect(dashboard.savedAt).toBe("ok");
  });
});
