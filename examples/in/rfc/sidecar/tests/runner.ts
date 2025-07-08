import { join } from "node:path";
import { ForstHTTPClient } from "./http_client";

interface TestResult {
  name: string;
  passed: boolean;
  error?: string;
  output?: string;
  errorOutput?: string;
}

// Get port from environment or default to 8080
const forstServerUrl = `http://localhost:${process.env.PORT || 8080}`;
const testsDir = __dirname;

async function runTest(testFile: string): Promise<TestResult> {
  const client = new ForstHTTPClient(forstServerUrl);
  const testPath = join(testsDir, testFile);

  try {
    // First check if server is healthy
    const isHealthy = await client.healthCheck();
    if (!isHealthy) {
      return {
        name: testFile,
        passed: false,
        error: "Forst HTTP server is not healthy",
      };
    }

    // Run the test via HTTP
    const response = await client.runTest(testPath);

    return {
      name: testFile,
      passed: response.success,
      output: response.output,
      error: response.error,
      errorOutput: response.error,
    };
  } catch (error) {
    return {
      name: testFile,
      passed: false,
      error: error instanceof Error ? error.message : String(error),
    };
  }
}

async function main() {
  console.log("Running Sidecar Integration Tests...\n");

  // Check if Forst server is running
  const client = new ForstHTTPClient(forstServerUrl);
  const isHealthy = await client.healthCheck();

  if (!isHealthy) {
    console.error("❌ Forst HTTP server is not running or not healthy");
    console.error("");
    console.error("To start the Forst development server:");
    console.error("  1. In the sidecar directory: ./start-forst-server.sh");
    console.error(
      "  2. Or manually: cd ../../../../../forst && ./bin/forst dev -port=8080"
    );
    console.error("");
    console.error(`The server will be available at: ${forstServerUrl}`);
    process.exit(1);
  }

  // Fetch and print available functions
  try {
    const resp = await fetch(`${forstServerUrl}/functions`);
    const data = await resp.json();
    if (data && data.result) {
      const functions = Array.isArray(data.result) ? data.result : [];
      console.log("Available Forst functions:");
      for (const fn of functions) {
        console.log(
          `  - ${fn.package}.${fn.name} (streaming: ${fn.supportsStreaming})`
        );
      }
      console.log("");
    }
  } catch (err) {
    console.warn("Could not fetch /functions endpoint:", err);
  }

  console.log("✅ Forst HTTP server is healthy\n");

  const tests = ["basic_communication.ft", "type_safety.ft"];

  for (const test of tests) {
    const result = await runTest(test);

    console.log(`Test: ${result.name}`);
    console.log(`Status: ${result.passed ? "PASS" : "FAIL"}`);
    if (result.error) {
      console.log(`Error: ${result.error}`);
    }
    if (result.output) {
      console.log(`Output: ${result.output}`);
    } else {
      console.log("Output: <none>");
    }

    if (result.errorOutput) {
      console.log(`Error Output: ${result.errorOutput}`);
    } else {
      console.log("Error Output: <none>");
    }

    console.log("");
  }
}

main().catch(console.error);
