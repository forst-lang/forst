/**
 * Example test double: same production import specifier, no HTTP.
 * Copied next to generated dist/ by TestGenerate_acceptance_testdataExampleRunsWithoutHTTP.
 */
import { withForstTestScope } from "./$testing.js";
import { Echo } from "./pkg/main.js";

const result = await withForstTestScope(
  {
    packages: {
      main: {
        Echo: async (input) => ({
          echo: "test:" + input.message,
          timestamp: 42,
        }),
      },
    },
  },
  async () => Echo({ message: "hello" })
);

if (result.echo !== "test:hello" || result.timestamp !== 42) {
  console.error("unexpected result", result);
  process.exit(1);
}
console.log("ok");
