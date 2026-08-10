import { Readable } from "node:stream";
import { describe, expect, it } from "bun:test";
import { readAuthHandoffFromStream } from "./invoke-auth-handoff";

describe("readAuthHandoffFromStream", () => {
  it("parses a single JSON line into generation and token bytes", async () => {
    const token = Buffer.from("abc123", "utf8").toString("base64url");
    const stream = Readable.from([
      JSON.stringify({ generation: 2, token }) + "\n",
    ]);
    const handoff = await readAuthHandoffFromStream(stream);
    expect(handoff.generation).toBe(2);
    expect(Buffer.from(handoff.token).toString("utf8")).toBe("abc123");
  });

  it("rejects malformed handoff payloads", async () => {
    const stream = Readable.from(['{"generation":1}\n']);
    const error = await readAuthHandoffFromStream(stream).catch(
      (caught: unknown) => caught
    );
    expect(error).toBeInstanceOf(Error);
    expect((error as Error).message).toMatch(/missing generation or token/);
  });
});
