import { request } from "node:http";

interface TestCase {
  name: string;
  test: string;
  data: unknown;
  expectSuccess: boolean;
}

async function runHttpTest(testCase: TestCase): Promise<void> {
  return new Promise((resolve, reject) => {
    const postData = JSON.stringify({
      test: testCase.test,
      data: testCase.data,
    });

    const req = request(
      {
        hostname: "localhost",
        port: 3001,
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Content-Length": Buffer.byteLength(postData),
        },
      },
      (res) => {
        let data = "";
        res.on("data", (chunk) => {
          data += chunk;
        });
        res.on("end", () => {
          try {
            const result = JSON.parse(data);
            const passed = testCase.expectSuccess
              ? result.success
              : !result.success;

            console.log(`Test: ${testCase.name}`);
            console.log(`Status: ${passed ? "PASS" : "FAIL"}`);
            console.log(`Response: ${JSON.stringify(result, null, 2)}`);
            console.log("");

            resolve();
          } catch (error) {
            reject(error);
          }
        });
      }
    );

    req.on("error", reject);
    req.write(postData);
    req.end();
  });
}

async function runAllHttpTests(): Promise<void> {
  console.log("Running HTTP Sidecar Tests...\n");

  const tests: TestCase[] = [
    {
      name: "Basic Communication",
      test: "basic_communication",
      data: { message: "hello world" },
      expectSuccess: true,
    },
    {
      name: "Error Handling - Invalid Input",
      test: "error_handling",
      data: { value: 15 }, // Should fail validation
      expectSuccess: false,
    },
    {
      name: "Type Safety - Valid User",
      test: "type_safety",
      data: { id: "user123" },
      expectSuccess: true,
    },
  ];

  for (const test of tests) {
    await runHttpTest(test);
  }
}

if (require.main === module) {
  runAllHttpTests().catch(console.error);
}

export { runHttpTest, runAllHttpTests };
