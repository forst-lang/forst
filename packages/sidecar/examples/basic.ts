// Basic example of using @forst/sidecar

import { autoStart } from "../src/index";

async function basicExample() {
  console.log("🚀 Starting basic Forst sidecar example...");

  try {
    // Create and start the sidecar
    const sidecar = await autoStart({
      forstDir: "../../forst", // Point to the main Forst directory
      port: 8080,
      logLevel: "info",
    });

    console.log("✅ Sidecar started successfully");

    // Discover available functions
    const functions = await sidecar.discoverFunctions();
    console.log(
      "📋 Discovered functions:",
      functions.map((f) => `${f.package}.${f.name}`)
    );

    // Example: Call a function if available
    if (functions.length > 0) {
      const firstFunction = functions[0];
      console.log(
        `🔧 Calling function: ${firstFunction.package}.${firstFunction.name}`
      );

      try {
        const result = await sidecar.invoke(
          firstFunction.package,
          firstFunction.name,
          {} // Empty args for this example
        );

        console.log("✅ Function call result:", result);
      } catch (error) {
        console.error("❌ Function call failed:", error);
      }
    }

    // Health check
    const isHealthy = await sidecar.healthCheck();
    console.log("🏥 Health check:", isHealthy ? "PASS" : "FAIL");

    // Keep the server running for a bit to see it in action
    console.log("⏳ Keeping server running for 10 seconds...");
    await new Promise((resolve) => setTimeout(resolve, 10000));
  } catch (error) {
    console.error("❌ Failed to start sidecar:", error);
    process.exit(1);
  }
}

// Run the example if this file is executed directly
if (require.main === module) {
  basicExample().catch(console.error);
}

export { basicExample };
