import { ForstClient } from "@forst/sidecar";

interface TestResult {
  name: string;
  passed: boolean;
  error?: string;
  output?: string;
  errorOutput?: string;
}

const serverProtocol = process.env.SERVER_PROTOCOL || "http";
const serverHost = process.env.HOST || "localhost";
const serverPort = process.env.PORT || "8081";
const serverUrl = `${serverProtocol}://${serverHost}:${serverPort}`;

async function runTest(
  client: ForstClient,
  test: Readonly<{
    fn: string;
    args?: any;
    want: {
      success?: boolean;
      output?: any;
      error?: string;
    };
  }>
): Promise<TestResult> {
  const result: TestResult = {
    name: test.fn,
    passed: false,
  };

  try {
    const response = await client.invoke(test.fn, test.args);

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

async function main() {
  console.log("Running Sidecar Integration Tests...\n");

  // Create HTTP client
  const client = new ForstClient({
    baseUrl: serverUrl,
    timeout: 10000,
    retries: 3,
  });

  // Check if server is running
  const isHealthy = await client.healthCheck();
  if (!isHealthy) {
    console.log("❌ Forst HTTP server is not healthy");
    console.log(
      "Note: This test requires a Forst dev server to be running on port 8081"
    );
    console.log(
      "Start it with: cd forst && go run ./cmd/forst dev -port 8081 -root examples/in/rfc/sidecar/tests"
    );
    return;
  }

  console.log("✅ Found existing Forst server, using it for tests");

  // Discover available functions
  const functions = await client.discoverFunctions();
  console.log("Available Forst functions:");
  for (const fn of functions) {
    console.log(
      `  - ${fn.package}.${fn.name} (streaming: ${fn.supportsStreaming})`
    );
  }
  console.log();

  console.log("✅ Forst HTTP server is healthy\n");

  // Define tests
  const tests = [
    {
      fn: "echo.Echo",
      args: { message: "Hello, Forst!" },
      want: {
        success: true,
        output: { echo: "Hello, Forst!", timestamp: 1234567890 },
      },
    },
    {
      fn: "typesafety.GetUserAge",
      args: {},
      want: {
        success: true,
        output: 25,
      },
    },
  ] as const;

  // Run tests
  const results: TestResult[] = [];
  for (const test of tests) {
    const result = await runTest(client, test);
    results.push(result);

    console.log(`Test: ${test.fn}`);
    console.log(`Status: ${result.passed ? "✅ PASS" : "❌ FAIL"}`);
    if (result.output) {
      console.log(`Output: ${result.output}`);
    }
    if (result.errorOutput) {
      console.log(`Error Output: ${result.errorOutput}`);
    }
    if (result.error) {
      console.log(`Error: ${result.error}`);
    }
    console.log();
  }

  // Summary
  const passed = results.filter((r) => r.passed).length;
  const total = results.length;

  if (passed === total) {
    console.log("✅ All tests passed!");
  } else {
    console.log(`❌ Some tests failed`);
    process.exit(1);
  }
}

main().catch((error) => {
  console.error("Test runner failed:", error);
  process.exit(1);
});
