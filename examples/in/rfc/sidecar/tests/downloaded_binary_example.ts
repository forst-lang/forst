import { ForstSidecar, createSidecar, ForstUtils } from "@forst/sidecar";
import { resolve } from "node:path";

async function runDownloadedBinaryExample() {
  console.log("🚀 Starting Downloaded Binary Example...\n");

  // First, ensure the Forst binary is downloaded
  console.log("📥 Downloading Forst binary...");
  const forstPath = await ForstUtils.ensureCompiler();
  console.log("✅ Forst binary available at:", forstPath);

  // Create sidecar instance that will use the downloaded binary
  const sidecar = createSidecar({
    mode: "development",
    port: 8082, // Use different port to avoid conflicts
    host: "localhost",
    logLevel: "info",
    rootDir: resolve(__dirname, "."), // Point to the tests directory
  });

  try {
    // Start the sidecar (this will use the downloaded binary)
    await sidecar.start();
    console.log("✅ Sidecar started successfully with downloaded binary");

    // Get server info
    const serverInfo = sidecar.getServerInfo();
    console.log("📊 Server Info:", serverInfo);

    // Discover available functions
    const functions = await sidecar.discoverFunctions();
    console.log("🔍 Available functions:");
    for (const fn of functions) {
      console.log(
        `  - ${fn.package}.${fn.name} (streaming: ${fn.supportsStreaming})`
      );
    }

    // Test function invocation
    console.log("\n🧪 Testing function invocation with downloaded binary...");

    // Test echo function
    const echoResult = await sidecar.invoke("echo", "Echo", {
      message: "Hello from downloaded binary!",
    });
    console.log("Echo result:", echoResult);

    // Test typesafety function
    const ageResult = await sidecar.invoke("typesafety", "GetUserAge", {});
    console.log("GetUserAge result:", ageResult);

    // Health check
    const isHealthy = await sidecar.healthCheck();
    console.log("Health check:", isHealthy ? "✅ Healthy" : "❌ Unhealthy");

    // Test streaming function (if available)
    console.log("\n🔄 Testing streaming function...");
    try {
      const streamingResult = await sidecar.invokeStreaming(
        "echo",
        "Echo",
        { message: "Streaming test" },
        (result) => {
          console.log("Streaming result:", result);
        }
      );
      console.log("Streaming test completed");
    } catch (error) {
      console.log(
        "Streaming not supported or failed:",
        error instanceof Error ? error.message : String(error)
      );
    }

    console.log("\n✅ All tests passed with downloaded binary!");
  } catch (error) {
    console.error("❌ Example failed:", error);
    throw error;
  } finally {
    // Stop the sidecar
    await sidecar.stop();
    console.log("🛑 Sidecar stopped");
  }
}

// Run the example
if (require.main === module) {
  runDownloadedBinaryExample().catch((error) => {
    console.error("Example failed:", error);
    process.exit(1);
  });
}

export { runDownloadedBinaryExample };
