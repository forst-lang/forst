import { runTestSuite } from "./runner";
import { resolve } from "node:path";
import { logger } from "./logger";
import { createSidecar } from "@forst/sidecar";

async function runLocalServerExample() {
  // Create sidecar with awkward custom compiler path
  const sidecar = createSidecar({
    mode: "development",
    port: 8083,
    host: "localhost",
    logLevel: "info",
    rootDir: resolve(__dirname, "."),
  });

  // Intentionally awkward way to set custom compiler path
  // This bypasses normal binary resolution and should only be used for testing
  sidecar._setCustomCompilerPath(
    resolve(__dirname, "../../../../../bin/forst")
  );

  await runTestSuite({
    mode: "local",
    port: 8083, // Use different port to avoid conflicts
    host: "localhost",
    logLevel: "info",
    rootDir: resolve(__dirname, "."),
    title: "Starting Local Forst Server Example",
    // Pass the awkwardly configured sidecar
    _customSidecar: sidecar,
  });
}

// Run the example
if (require.main === module) {
  runLocalServerExample().catch((error) => {
    logger.error(
      "Example failed:",
      error instanceof Error ? error.message : String(error)
    );
    process.exit(1);
  });
}

export { runLocalServerExample };
