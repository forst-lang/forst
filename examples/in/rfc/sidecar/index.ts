import { createServer } from "node:http";
import { ForstClient } from "../../client"; // Adjust path as needed

const port = 3000;

// Initialize Forst client (connects to dev server/sidecar)
const forst = new ForstClient({
  baseUrl: "http://localhost:8081",
  timeout: 30000,
  retries: 3,
});

// Start the server
async function startServer() {
  const server = createServer(async (req, res) => {
    if (req.method === "POST" && req.url === "/") {
      let body = "";
      req.on("data", (chunk) => {
        body += chunk.toString();
      });

      req.on("end", async () => {
        try {
          const input = JSON.parse(body);
          // Call the validation function using the new client
          const result = await forst.validation.ValidateUser(input);

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

    // Test the client
    try {
      // Test a simple function call
      const result = await forst.echo.Echo({ message: "Hello from RFC!" });
      console.log("Test result:", result);
    } catch (err) {
      console.error("Error:", err);
    } finally {
      server.close();
    }
  });
}

startServer();
