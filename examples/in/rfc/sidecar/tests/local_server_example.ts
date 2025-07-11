import { runTestSuite } from "./runner";
import { resolve } from "node:path";
import { logger } from "./logger";

async function runLocalServerExample() {
  await runTestSuite({
    mode: "local",
    port: 8083, // Use different port to avoid conflicts
    host: "localhost",
    logLevel: "info",
    rootDir: resolve(__dirname, "."),
    title: "🚀 Starting Local Forst Server Example",
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
