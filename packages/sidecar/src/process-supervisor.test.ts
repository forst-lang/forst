import { describe, expect, test } from "bun:test";
import { ProcessSupervisor } from "./process-supervisor";

describe("ProcessSupervisor", () => {
  test("starts in stopped state with no child", () => {
    const supervisor = new ProcessSupervisor("/usr/bin/forst", "127.0.0.1", 6320);
    expect(supervisor.processStatus).toBe("stopped");
    expect(supervisor.child).toBeNull();
  });

  test("stop is a no-op when no child is running", async () => {
    const supervisor = new ProcessSupervisor("/usr/bin/forst", "127.0.0.1", 6320);
    await supervisor.stop();
    expect(supervisor.processStatus).toBe("stopped");
    expect(supervisor.child).toBeNull();
  });

  test("setProcessStatus updates observable status", () => {
    const supervisor = new ProcessSupervisor("/usr/bin/forst", "127.0.0.1", 6320);
    supervisor.setProcessStatus("starting");
    expect(supervisor.processStatus).toBe("starting");
    supervisor.setProcessStatus("running");
    expect(supervisor.processStatus).toBe("running");
  });
});
