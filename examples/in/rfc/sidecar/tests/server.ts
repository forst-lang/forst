import { createServer, IncomingMessage, ServerResponse } from "node:http";
import { spawn } from "node:child_process";
import { resolve } from "node:path";

const forstDir = resolve(__dirname, "../../../../../forst/cmd/forst");

interface SidecarRequest {
  test: string;
  data: unknown;
}

interface SidecarResponse {
  success: boolean;
  data?: unknown;
  error?: string;
}

async function handleForstRequest(
  testFile: string,
  input: string
): Promise<SidecarResponse> {
  return new Promise((resolve) => {
    const forstProc = spawn("go", ["run", ".", "run", testFile], {
      stdio: ["pipe", "pipe", "pipe"],
      timeout: 3000,
      cwd: forstDir,
    });

    let output = "";
    let errorOutput = "";

    forstProc.stdout?.on("data", (chunk) => {
      output += chunk.toString();
    });

    forstProc.stderr?.on("data", (chunk) => {
      errorOutput += chunk.toString();
    });

    forstProc.on("close", (code) => {
      if (code === 0) {
        try {
          const data = JSON.parse(output);
          resolve({ success: true, data });
        } catch {
          resolve({ success: true, data: output.trim() });
        }
      } else {
        resolve({
          success: false,
          error: errorOutput.trim() || "Forst execution failed",
        });
      }
    });

    forstProc.stdin?.write(input);
    forstProc.stdin?.end();
  });
}

async function handleRequest(
  req: IncomingMessage,
  res: ServerResponse
): Promise<void> {
  if (req.method !== "POST") {
    res.statusCode = 405;
    res.end(JSON.stringify({ error: "Method not allowed" }));
    return;
  }

  let body = "";
  req.on("data", (chunk) => {
    body += chunk.toString();
  });

  req.on("end", async () => {
    try {
      const request: SidecarRequest = JSON.parse(body);
      const testFile = resolve(__dirname, `${request.test}.ft`);

      const result = await handleForstRequest(
        testFile,
        JSON.stringify(request.data)
      );

      res.setHeader("Content-Type", "application/json");
      res.end(JSON.stringify(result));
    } catch (error) {
      res.statusCode = 400;
      res.end(JSON.stringify({ error: "Invalid request format" }));
    }
  });
}

const server = createServer(handleRequest);
const port = 3001;

server.listen(port, () => {
  console.log(`Sidecar test server running on http://localhost:${port}`);
  console.log("Available tests: echo, error_handling, type_safety");
});

export { handleForstRequest };
