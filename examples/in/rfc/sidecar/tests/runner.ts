import { spawn } from "node:child_process";
import { join } from "node:path";

interface TestResult {
  name: string;
  passed: boolean;
  error?: string;
  output?: string;
}

const forstDir = join(__dirname, "../../../../../forst/cmd/forst");
const testsDir = __dirname;

async function runTest(testFile: string): Promise<TestResult> {
  return new Promise((resolve) => {
    const testPath = join(testsDir, testFile);
    const forstProc = spawn("go", ["run", ".", "run", testPath], {
      stdio: ["pipe", "pipe", "pipe"],
      timeout: 3000,
      cwd: forstDir,
    });

    let output = "";
    let errorOutput = "";

    forstProc.stdout?.on("data", (chunk: Buffer) => {
      output += chunk.toString();
    });

    forstProc.stderr?.on("data", (chunk: Buffer) => {
      errorOutput += chunk.toString();
    });

    forstProc.on("close", (code: number) => {
      resolve({
        name: testFile,
        passed: code === 0,
        error: code !== 0 ? errorOutput : undefined,
        output: code === 0 ? output : undefined,
      });
    });

    forstProc.on("error", (error: Error) => {
      resolve({
        name: testFile,
        passed: false,
        error: error.message,
      });
    });
  });
}

async function main() {
  console.log("Running Sidecar Integration Tests...\n");

  const tests = ["basic_communication.ft", "type_safety.ft"];

  for (const test of tests) {
    const result = await runTest(test);

    console.log(`Test: ${result.name}`);
    console.log(`Status: ${result.passed ? "PASS" : "FAIL"}`);
    if (result.error) {
      console.log(`Error: ${result.error}`);
    }
    if (result.output) {
      console.log(`Output: ${result.output}`);
    } else {
      console.log("Output: <none>");
    }
    console.log("");
  }
}

main().catch(console.error);
