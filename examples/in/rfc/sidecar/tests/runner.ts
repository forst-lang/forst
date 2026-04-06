import { ForstSidecar, ForstUtils } from "@forst/sidecar";
import { runnerLogger } from "./logger";

interface TestRunnerConfig {
  mode: "local" | "downloaded";
  port: number;
  host: string;
  logLevel: "info" | "debug" | "warn" | "error";
  rootDir: string;
  title: string;
  _customSidecar?: ForstSidecar; // Intentionally awkward - don't use this normally
}

interface TestResult {
  name: string;
  passed: boolean;
  error?: string;
  output?: string;
  errorOutput?: string;
}

async function runTest(
  sidecar: ForstSidecar,
  test: Readonly<{
    package: string;
    function: string;
    args?: any[];
    want: {
      success?: boolean;
      output?: any;
      error?: string;
    };
  }>
): Promise<TestResult> {
  const result: TestResult = {
    name: `${test.package}.${test.function}`,
    passed: false,
  };

  try {
    const response = await sidecar.invoke(
      test.package,
      test.function,
      test.args
    );

    // Debug: Log the full response structure
    runnerLogger.debug(
      `Full response for ${test.package}.${test.function}:`,
      JSON.stringify(response, null, 2)
    );

    if (
      test.want.success !== undefined &&
      response.success !== test.want.success
    ) {
      result.passed = false;
      result.error = `Expected success: ${test.want.success}, got: ${response.success}`;
      return result;
    }

    if (test.want.output !== undefined) {
      if (
        JSON.stringify(response.result) !== JSON.stringify(test.want.output)
      ) {
        result.passed = false;
        result.error = `Expected output: ${JSON.stringify(
          test.want.output
        )}, got: ${JSON.stringify(response.result)}`;
        result.output = JSON.stringify(response.result);
        return result;
      }
    }

    if (test.want.error !== undefined) {
      if (!response.error || !response.error.includes(test.want.error)) {
        result.passed = false;
        result.error = `Expected error to contain: ${test.want.error}, got: ${response.error}`;
        result.errorOutput = response.error;
        return result;
      }
    }

    result.passed = true;
    result.output = JSON.stringify(response.result);
    return result;
  } catch (error) {
    result.passed = false;
    result.error = `Test failed with exception: ${error}`;
    return result;
  }
}

async function runTestSuite(config: TestRunnerConfig): Promise<boolean> {
  runnerLogger.info(`🚀 ${config.title}...`);

  let sidecar: ForstSidecar;
  let cleanupHandler: (() => void) | undefined;

  // Check if port is available
  const checkPortAvailable = async (port: number): Promise<boolean> => {
    return new Promise((resolve) => {
      const net = require("net");
      const server = net.createServer();

      server.listen(port, () => {
        server.close();
        resolve(true);
      });

      server.on("error", () => {
        resolve(false);
      });
    });
  };

  // Set up process signal handlers for cleanup
  const setupCleanup = (sidecarInstance: ForstSidecar) => {
    const cleanup = async (exitCode: number) => {
      runnerLogger.info("🛑 Received interrupt signal, cleaning up...");
      try {
        await sidecarInstance.stop();
        runnerLogger.info("✅ Forst server stopped successfully");
      } catch (error) {
        runnerLogger.error("❌ Failed to stop Forst server:", error);
      }
      process.exit(exitCode);
    };

    const onSignal = () => {
      void cleanup(0).catch((err) => {
        runnerLogger.error("cleanup failed:", err);
        process.exit(1);
      });
    };

    // Handle various interrupt signals
    process.on("SIGINT", onSignal); // Ctrl+C
    process.on("SIGTERM", onSignal); // Termination request
    process.on("SIGQUIT", onSignal); // Quit request
    process.on("SIGUSR1", onSignal); // User defined signal 1
    process.on("SIGUSR2", onSignal); // User defined signal 2

    const onException = (error: Error) => {
      runnerLogger.error("❌ Uncaught exception:", error);
      void cleanup(1).catch(() => process.exit(1));
    };

    const onRejection = (reason: unknown, promise: Promise<unknown>) => {
      runnerLogger.error(
        "❌ Unhandled rejection at:",
        promise,
        "reason:",
        reason
      );
      void cleanup(1).catch(() => process.exit(1));
    };

    process.on("uncaughtException", onException);
    process.on("unhandledRejection", onRejection);

    return () => {
      process.off("SIGINT", onSignal);
      process.off("SIGTERM", onSignal);
      process.off("SIGQUIT", onSignal);
      process.off("SIGUSR1", onSignal);
      process.off("SIGUSR2", onSignal);
      process.off("uncaughtException", onException);
      process.off("unhandledRejection", onRejection);
    };
  };

  // Initialize based on mode
  if (config.mode === "downloaded") {
    runnerLogger.info("📥 Downloading Forst binary...");
    const forstPath = await ForstUtils.ensureCompiler();
    runnerLogger.info(`✅ Forst binary available at: ${forstPath}`);
  }

  // Check if port is available
  const isPortAvailable = await checkPortAvailable(config.port);
  if (!isPortAvailable) {
    runnerLogger.warn(
      `⚠️  Port ${config.port} is already in use. This might cause issues.`
    );
  }

  // Create sidecar instance
  if (config._customSidecar) {
    // Intentionally awkward way to use custom sidecar
    runnerLogger.warn(
      "⚠️  Using custom sidecar - this is intentionally awkward and not recommended for normal use"
    );
    sidecar = config._customSidecar;
  } else {
    // Normal way to create sidecar
    sidecar = new ForstSidecar({
      mode: "development",
      port: config.port,
      host: config.host,
      logLevel: config.logLevel,
      rootDir: config.rootDir,
    });
  }

  try {
    // Set up cleanup handlers
    cleanupHandler = setupCleanup(sidecar);

    // Start the sidecar
    await sidecar.start();
    runnerLogger.info("✅ Sidecar started successfully");

    // Get server info
    const serverInfo = sidecar.getServerInfo();
    runnerLogger.info("📊 Server Info:", serverInfo);

    // Discover available functions
    const functions = await sidecar.discoverFunctions();
    runnerLogger.info("🔍 Available functions:");
    for (const fn of functions) {
      runnerLogger.info(
        `  - ${fn.package}.${fn.name} (streaming: ${fn.supportsStreaming})`
      );
    }

    // Health check
    const isHealthy = await sidecar.healthCheck();
    runnerLogger.info(
      "Health check:",
      isHealthy ? "✅ Healthy" : "❌ Unhealthy"
    );

    // Abort if health check fails
    if (!isHealthy) {
      runnerLogger.error("❌ Health check failed. Aborting test run.");
      throw new Error("Sidecar health check failed");
    }

    // Define tests
    const tests = [
      {
        package: "echo",
        function: "Echo",
        args: [{ message: "Hello, Forst!" }],
        want: {
          success: true,
          output: { echo: "Hello, Forst!", timestamp: 1234567890 },
        },
      },
      {
        package: "typesafety",
        function: "GetUserAge",
        args: [],
        want: {
          success: true,
          output: 25,
        },
      },
    ] as const;

    // Run tests
    runnerLogger.info("🧪 Running tests...");
    const results: TestResult[] = [];
    for (const test of tests) {
      const result = await runTest(sidecar, test);
      results.push(result);

      runnerLogger.info(`Test: ${test.package}.${test.function}`);
      runnerLogger.info(`Status: ${result.passed ? "✅ PASS" : "❌ FAIL"}`);
      if (result.output) {
        runnerLogger.info(`Output: ${result.output}`);
      }
      if (result.errorOutput) {
        runnerLogger.info(`Error: ${result.errorOutput}`);
      }
      if (result.error) {
        runnerLogger.info(`Error: ${result.error}`);
      }
    }

    // Summary
    const passed = results.filter((r) => r.passed).length;
    const total = results.length;

    // Print summary table
    runnerLogger.info("Test Summary:");
    for (const result of results) {
      runnerLogger.info(
        `- ${result.name}: ${result.passed ? "✅ PASS" : "❌ FAIL"}`
      );
    }

    if (passed === total) {
      runnerLogger.info("✅ All tests passed!");
      return true;
    } else {
      runnerLogger.info("❌ Some tests failed");
      return false;
    }
  } catch (error) {
    runnerLogger.error(
      "❌ Test suite failed:",
      error instanceof Error ? error.message : String(error)
    );
    // Ensure sidecar is stopped even on error
    try {
      await sidecar.stop();
      runnerLogger.info("🛑 Sidecar stopped after error");
    } catch (stopError) {
      runnerLogger.error(
        "Failed to stop sidecar:",
        stopError instanceof Error ? stopError.message : String(stopError)
      );
    }
    throw error;
  } finally {
    // Clean up signal handlers
    if (cleanupHandler) {
      cleanupHandler();
    }

    // Stop the sidecar
    try {
      await sidecar.stop();
      runnerLogger.info("🛑 Sidecar stopped");
    } catch (stopError) {
      runnerLogger.error(
        "Failed to stop sidecar in finally:",
        stopError instanceof Error ? stopError.message : String(stopError)
      );
    }
  }
}

export { runTestSuite, TestRunnerConfig };
