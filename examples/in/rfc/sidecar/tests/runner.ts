import { ForstSidecar, createSidecar, ForstUtils } from "@forst/sidecar";
import { resolve } from "node:path";
import { runnerLogger, testLogger } from "./logger";

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
    args?: any;
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
    const cleanup = async () => {
      runnerLogger.info("🛑 Received interrupt signal, cleaning up...");
      try {
        await sidecarInstance.stop();
        runnerLogger.info("✅ Forst server stopped successfully");
      } catch (error) {
        runnerLogger.error("❌ Failed to stop Forst server:", error);
      }
      process.exit(0);
    };

    // Handle various interrupt signals
    process.on("SIGINT", cleanup); // Ctrl+C
    process.on("SIGTERM", cleanup); // Termination request
    process.on("SIGQUIT", cleanup); // Quit request
    process.on("SIGUSR1", cleanup); // User defined signal 1
    process.on("SIGUSR2", cleanup); // User defined signal 2

    // Handle uncaught exceptions and unhandled rejections
    process.on("uncaughtException", (error) => {
      runnerLogger.error("❌ Uncaught exception:", error);
      cleanup();
    });

    process.on("unhandledRejection", (reason, promise) => {
      runnerLogger.error(
        "❌ Unhandled rejection at:",
        promise,
        "reason:",
        reason
      );
      cleanup();
    });

    return () => {
      process.off("SIGINT", cleanup);
      process.off("SIGTERM", cleanup);
      process.off("SIGQUIT", cleanup);
      process.off("SIGUSR1", cleanup);
      process.off("SIGUSR2", cleanup);
      process.off("uncaughtException", cleanup);
      process.off("unhandledRejection", cleanup);
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
    sidecar = createSidecar({
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
        args: { message: "Hello, Forst!" },
        want: {
          success: true,
          output: { echo: "Hello, Forst!", timestamp: 1234567890 },
        },
      },
      {
        package: "typesafety",
        function: "GetUserAge",
        args: {},
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

// Default configuration for backward compatibility
const defaultConfig: TestRunnerConfig = {
  mode: "local",
  port: 8083, // Use different port to avoid conflicts with existing processes
  host: "localhost",
  logLevel: "info",
  rootDir: resolve(__dirname, "."),
  title: "Running Sidecar Integration Tests",
};

async function main() {
  // Check if we're running in a specific mode
  const mode = process.env.FORST_MODE as "local" | "downloaded" | undefined;
  const port = process.env.FORST_PORT ? parseInt(process.env.FORST_PORT) : 8081;

  const config: TestRunnerConfig = {
    ...defaultConfig,
    mode: mode || "local",
    port,
    title:
      mode === "downloaded"
        ? "Running Downloaded Binary Example"
        : "Running Local Forst Server Example",
  };

  try {
    const success = await runTestSuite(config);
    process.exit(success ? 0 : 1);
  } catch (error) {
    runnerLogger.error(
      "Test runner failed:",
      error instanceof Error ? error.message : String(error)
    );
    process.exit(1);
  }
}

// Run the test suite
if (require.main === module) {
  main().catch((error) => {
    runnerLogger.error(
      "Test runner failed:",
      error instanceof Error ? error.message : String(error)
    );
    // Exit with code 1 if tests failed, 0 otherwise
    const exitCode =
      error instanceof Error && error.message === "Some tests failed" ? 1 : 1;
    process.exit(exitCode);
  });
}

export { runTestSuite, TestRunnerConfig };
