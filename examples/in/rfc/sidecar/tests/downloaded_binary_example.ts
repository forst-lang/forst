import { runTestSuite } from "./runner";
import { resolve } from "node:path";
import { logger } from "./logger";

async function runDownloadedBinaryExample() {
  await runTestSuite({
    mode: "downloaded",
    port: 8084, // Use different port to avoid conflicts
    host: "localhost",
    logLevel: "info",
    rootDir: resolve(__dirname, "."),
    title: "🚀 Starting Downloaded Binary Example",
  });
}

// Run the example
if (require.main === module) {
  runDownloadedBinaryExample().catch((error) => {
    logger.error(
      "Example failed:",
      error instanceof Error ? error.message : String(error)
    );
    process.exit(1);
  });
}

export { runDownloadedBinaryExample };
