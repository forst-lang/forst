import { ForstSidecar, createSidecar } from "@forst/sidecar";
import { resolve } from "node:path";

async function runLocalServerExample() {
  console.log("🚀 Starting Local Forst Server Example...\n");

  // Create sidecar instance with local configuration
  const sidecar = createSidecar({
    mode: "development",
    port: 8081,
    host: "localhost",
    logLevel: "info",
    rootDir: resolve(__dirname, "."), // Point to the tests directory
  });

  try {
    // Start the sidecar (this will start the Forst dev server)
    await sidecar.start();
    console.log("✅ Sidecar started successfully");

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
    console.log("\n🧪 Testing function invocation...");

    // Test echo function
    const echoResult = await sidecar.invoke("echo", "Echo", {
      message: "Hello from local server!",
    });
    console.log("Echo result:", echoResult);

    // Test typesafety function
    const ageResult = await sidecar.invoke("typesafety", "GetUserAge", {});
    console.log("GetUserAge result:", ageResult);

    // Health check
    const isHealthy = await sidecar.healthCheck();
    console.log("Health check:", isHealthy ? "✅ Healthy" : "❌ Unhealthy");

    console.log("\n✅ All tests passed!");
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
  runLocalServerExample().catch((error) => {
    console.error("Example failed:", error);
    process.exit(1);
  });
}

export { runLocalServerExample };
