import { createServer } from "node:http";
import { ForstSidecar } from "@forst/sidecar";

const port = 3000;

// Initialize Forst sidecar
const sidecar = new ForstSidecar({
  port: 8081,
  mode: "development",
});

// Start the sidecar
async function startServer() {
  try {
    await sidecar.start();

    const server = createServer(async (req, res) => {
      if (req.method === "POST" && req.url === "/") {
        let body = "";
        req.on("data", (chunk) => {
          body += chunk.toString();
        });

        req.on("end", async () => {
          try {
            const input = JSON.parse(body);
            const client = sidecar.getClient();
            const result = await client.invokeFunction(
              "validation",
              "ValidateUser",
              input
            );

            res.setHeader("Content-Type", "application/json");
            res.end(JSON.stringify(result));
          } catch (error) {
            res.statusCode = 500;
            res.end(
              JSON.stringify({
                success: false,
                error:
                  error instanceof Error ? error.message : "Validation failed",
              })
            );
          }
        });
      } else {
        res.statusCode = 404;
        res.end("Not found");
      }
    });

    server.listen(port, async () => {
      console.log(`Server running at http://localhost:${port}`);

      // Test the sidecar
      try {
        const client = sidecar.getClient();
        const functions = await client.discoverFunctions();
        console.log("Available functions:", functions);

        // Test a simple function call
        const result = await client.invoke("echo.Echo", {
          message: "Hello from RFC!",
        });
        console.log("Test result:", result);
      } catch (err) {
        console.error("Error:", err);
      } finally {
        server.close();
        await sidecar.stop();
      }
    });
  } catch (error) {
    console.error("Failed to start server:", error);
    process.exit(1);
  }
}

startServer();
