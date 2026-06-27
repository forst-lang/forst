import { logger } from "./logger";

import type { ForstConfig } from "./types";
import { ForstSidecar } from "./forst-sidecar";

/**
 * Auto-start function for zero-config usage
 */
export async function autoStart(
  config?: Partial<ForstConfig>
): Promise<ForstSidecar> {
  const sidecar = new ForstSidecar(config);
  await sidecar.start();

  process.on("SIGINT", async () => {
    logger.info("\n🛑 Received SIGINT, shutting down gracefully...");
    await sidecar.stop();
    process.exit(0);
  });

  process.on("SIGTERM", async () => {
    logger.info("\n🛑 Received SIGTERM, shutting down gracefully...");
    await sidecar.stop();
    process.exit(0);
  });

  return sidecar;
}
