// Test client for the new package-based Forst API
const { ForstClient } = require("./forst");

async function testForstAPI() {
  const client = new ForstClient({
    baseUrl: "http://localhost:8080",
    timeout: 10000,
    retries: 3,
  });

  try {
    // Health check
    console.log("Testing health check...");
    const healthy = await client.healthCheck();
    console.log("Health check:", healthy ? "PASS" : "FAIL");

    // Discover functions
    console.log("\nDiscovering functions...");
    const functions = await client.discoverFunctions();
    console.log("Discovered functions:", functions.length);
    functions.forEach((fn) => {
      console.log(
        `  - ${fn.package}.${fn.name} (streaming: ${fn.supportsStreaming})`
      );
    });

    // Test regular function
    if (functions.some((f) => f.name === "ProcessData")) {
      console.log("\nTesting regular function...");
      const result = await client.invoke("example", "ProcessData", {
        input: "test data",
        value: 42,
      });
      console.log("Regular function result:", result);
    }

    // Test streaming function
    if (functions.some((f) => f.name === "ProcessStream")) {
      console.log("\nTesting streaming function...");
      await client.invokeStreaming(
        "example",
        "ProcessStream",
        {
          input: "streaming test data",
          count: 3,
        },
        (result) => {
          console.log("Streaming result:", result);
        }
      );
    }
  } catch (error) {
    console.error("Test failed:", error.message);
    process.exit(1);
  }
}

// Run the test
testForstAPI()
  .then(() => {
    console.log("\nAll tests completed successfully!");
  })
  .catch((error) => {
    console.error("Test suite failed:", error);
    process.exit(1);
  });
