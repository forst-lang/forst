import { expect, test } from "bun:test";
import { fetchWithRetry } from "./http.js";

test("fetchWithRetry retries until success on 503", async () => {
  let calls = 0;
  const fetchImpl: typeof fetch = async () => {
    calls++;
    if (calls < 3) {
      return new Response(null, { status: 503, statusText: "Service Unavailable" });
    }
    return new Response("ok", { status: 200 });
  };

  const res = await fetchWithRetry(fetchImpl, "https://example.com", undefined, 5);
  expect(res.ok).toBe(true);
  expect(calls).toBe(3);
});
