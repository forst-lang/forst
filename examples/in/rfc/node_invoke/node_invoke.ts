import { spawn, SpawnOptions } from "node:child_process";
import { IncomingMessage, ServerResponse } from "node:http";
import { resolve } from "node:path";

interface ValidationResult {
  success: boolean;
  data?: unknown;
  error?: string;
}

interface SpawnError extends Error {
  code?: string;
  path?: string;
  syscall?: string;
  spawnargs?: string[];
}

const forstDir = resolve(__dirname, "../../../../forst/cmd/forst");
const inputFile = resolve(__dirname, "../../../in/input_validation.skip.ft");

async function handle(
  req: IncomingMessage,
  res: ServerResponse,
): Promise<void> {
  const spawnOpts = {
    stdio: ["pipe", "pipe", "pipe"], // Capture stderr
    timeout: 5000, // 5 second timeout
    cwd: forstDir,
    env: {
      ...process.env,
      PATH: process.env.PATH,
    },
  } as const satisfies Readonly<SpawnOptions>;

  const forstProc = spawn("go", ["run", ".", "run", inputFile], spawnOpts);

  // Handle process errors
  forstProc.on("error", (err: SpawnError) => {
    console.error("Failed to start Forst process:", {
      error: err,
      code: err.code,
      path: err.path,
      syscall: err.syscall,
      spawnargs: err.spawnargs,
    });
    res.statusCode = 500;
    res.end(JSON.stringify({ success: false, error: "Internal server error" }));
  });

  // Handle timeout
  const timeout = setTimeout(() => {
    forstProc.kill();
    res.statusCode = 504;
    res.end(JSON.stringify({ success: false, error: "Validation timeout" }));
  }, spawnOpts.timeout);

  // Pipe the HTTP body to Forst binary
  if (!forstProc.stdin) {
    res.statusCode = 500;
    res.end(
      JSON.stringify({ success: false, error: "Failed to create stdin pipe" }),
    );
    return;
  }
  req.pipe(forstProc.stdin);

  let output = "";
  let errorOutput = "";
  if (!forstProc.stdout) {
    res.statusCode = 500;
    res.end(
      JSON.stringify({ success: false, error: "Failed to create stdout pipe" }),
    );
    return;
  }
  forstProc.stdout.on("data", (chunk: Buffer) => {
    output += chunk.toString();
  });

  if (!forstProc.stderr) {
    res.statusCode = 500;
    res.end(
      JSON.stringify({ success: false, error: "Failed to create stderr pipe" }),
    );
    return;
  }
  forstProc.stderr.on("data", (chunk: Buffer) => {
    errorOutput += chunk.toString();
  });

  forstProc.on("close", (code: number | null) => {
    clearTimeout(timeout);

    if (code === null) {
      console.error("Process terminated unexpectedly", {
        stdout: output,
        stderr: errorOutput,
      });
      res.statusCode = 500;
      return res.end(
        JSON.stringify({
          success: false,
          error: "Process terminated unexpectedly",
        }),
      );
    }

    if (code !== 0) {
      console.error("Process exited with non-zero code", {
        code,
        stdout: output,
        stderr: errorOutput,
      });
      res.statusCode = 400;
      return res.end(
        JSON.stringify({
          success: false,
          error: "Validation failed",
          details: errorOutput || output,
        }),
      );
    }

    try {
      const result: ValidationResult = {
        success: true,
        data: JSON.parse(output),
      };
      res.setHeader("Content-Type", "application/json");
      res.end(JSON.stringify(result));
    } catch (err) {
      console.error("Failed to parse validation result", {
        error: err,
        stdout: output,
        stderr: errorOutput,
      });
      res.statusCode = 500;
      res.end(
        JSON.stringify({
          success: false,
          error: "Failed to parse validation result",
          details: errorOutput || output,
        }),
      );
    }
  });
}

export { handle };
