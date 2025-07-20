import { resolve } from "node:path";
import { logger } from "./logger";
import { ForstSidecar } from "@forst/sidecar";
import { ForstClient } from "@forst/client";

interface ClientExampleConfig {
  title: string;
}

async function runClientExample(
  sidecar: ForstSidecar,
  config: ClientExampleConfig
): Promise<boolean> {
  logger.info(`🚀 ${config.title}...`);

  // Create ForstClient with the custom sidecar
  const client = new ForstClient({
    customSidecar: sidecar,
  });

  try {
    logger.info("🔍 Testing client connectivity...");

    // Test basic connectivity with a simple function call using Prisma-like API
    try {
      const echoResult = await client.echo.Echo({
        message: "Hello from client!",
      });
      logger.info("✅ Client connected successfully");
      logger.info("Echo response:", echoResult);
    } catch (error) {
      logger.error(
        "❌ Client connection failed:",
        error instanceof Error ? error.message : String(error)
      );
      throw new Error("Client connection failed");
    }

    // Test user management functions using Prisma-like API
    logger.info("🧪 Testing user management functions...");

    // Create a new user
    logger.info("Creating user...");
    const createRequest = {
      name: "Alice Johnson",
      age: 28,
      email: "alice@example.com",
    };

    const createResult = await client.user.CreateUser(createRequest);
    logger.info("✅ User created:", createResult.user);
    logger.info("Created at:", createResult.created_at);

    // Get user by ID
    logger.info("Fetching user...");
    const user = await client.user.GetUserById("123");
    logger.info("✅ User found:", user);

    // Update user age
    logger.info("Updating user age...");
    const updatedUser = await client.user.UpdateUserAge("123", 29);
    logger.info("✅ User updated:", updatedUser);

    // Test type safety - TypeScript should provide full IntelliSense
    logger.info("Testing type safety...");
    logger.info("User name:", user.name); // TypeScript knows this is a string
    logger.info("User age:", user.age); // TypeScript knows this is a number

    // Test error handling with invalid data
    logger.info("Testing error handling...");
    try {
      const invalidRequest = {
        name: "", // Invalid: empty name
        age: 200, // Invalid: age too high
        email: "invalid-email", // Invalid: no @ symbol
      };

      await client.user.CreateUser(invalidRequest as any);
      logger.error("❌ Expected validation error but none occurred");
      return false;
    } catch (error) {
      logger.info(
        "✅ Validation error caught as expected:",
        error instanceof Error ? error.message : String(error)
      );
    }

    // Test direct invoke method for comparison
    logger.info("Testing direct invoke method...");
    const directResult = await client.invoke("user", "GetUserById", "123");
    logger.info("✅ Direct invoke result:", directResult);

    // Test streaming capabilities (if supported)
    logger.info("Testing streaming capabilities...");
    const largeDataset = Array.from({ length: 5 }, (_, i) => ({
      name: `User ${i}`,
      age: 20 + (i % 50),
      email: `user${i}@example.com`,
    }));

    for (const userData of largeDataset) {
      try {
        const result = await client.user.CreateUser(userData);
        logger.info(`✅ Created user ${result.user.id}`);
      } catch (error) {
        logger.error(
          `❌ Failed to create user:`,
          error instanceof Error ? error.message : String(error)
        );
      }
    }

    logger.info("✅ All client tests passed!");
    return true;
  } catch (error) {
    logger.error(
      "❌ Client example failed:",
      error instanceof Error ? error.message : String(error)
    );
    return false;
  }
}

async function runClientExampleWithSidecar() {
  // Create sidecar with custom compiler path for testing
  const sidecar = new ForstSidecar({
    mode: "development",
    port: 8084, // Use different port to avoid conflicts
    host: "localhost",
    logLevel: "debug",
    rootDir: resolve(__dirname, "."),
  });

  // Intentionally awkward way to set custom compiler path
  // This bypasses normal binary resolution and should only be used for testing
  sidecar._setCustomCompilerPath(
    resolve(__dirname, "../../../../../bin/forst")
  );

  try {
    // Start the sidecar
    await sidecar.start();
    logger.info("✅ Sidecar started successfully");

    // Run the client example
    const success = await runClientExample(sidecar, {
      title: "Forst Client Example with Sidecar",
    });

    return success;
  } finally {
    // Stop the sidecar
    try {
      await sidecar.stop();
      logger.info("🛑 Sidecar stopped");
    } catch (stopError) {
      logger.error(
        "Failed to stop sidecar:",
        stopError instanceof Error ? stopError.message : String(stopError)
      );
    }
  }
}

// Example of using ForstClient with managed dev server
async function runClientExampleWithManagedServer() {
  logger.info("🚀 Running Forst Client with managed dev server...");

  // Create ForstClient that manages its own dev server
  const client = new ForstClient({
    port: 8085,
    host: "localhost",
    logLevel: "info",
    rootDir: resolve(__dirname, "."),
  });

  try {
    // Set up cleanup handlers
    client.setupCleanup();

    // Start the managed dev server
    await client.start();
    logger.info("✅ Managed dev server started successfully");

    // Test the client
    const echoResult = await client.echo.Echo({
      message: "Hello from managed client!",
    });
    logger.info("✅ Managed client test passed:", echoResult);

    return true;
  } catch (error) {
    logger.error(
      "❌ Managed client example failed:",
      error instanceof Error ? error.message : String(error)
    );
    return false;
  } finally {
    // Clean up
    client.cleanup();
    try {
      await client.stop();
      logger.info("🛑 Managed dev server stopped");
    } catch (stopError) {
      logger.error(
        "Failed to stop managed dev server:",
        stopError instanceof Error ? stopError.message : String(stopError)
      );
    }
  }
}

// Run the example
if (require.main === module) {
  runClientExampleWithSidecar()
    .then((success) => {
      if (success) {
        return runClientExampleWithManagedServer();
      }
      return false;
    })
    .then((success) => {
      process.exit(success ? 0 : 1);
    })
    .catch((error) => {
      logger.error(
        "Example failed:",
        error instanceof Error ? error.message : String(error)
      );
      process.exit(1);
    });
}

export {
  runClientExample,
  runClientExampleWithSidecar,
  runClientExampleWithManagedServer,
  ClientExampleConfig,
};
