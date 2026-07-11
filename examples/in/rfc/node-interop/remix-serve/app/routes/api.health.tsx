import type { LoaderFunctionArgs } from "@remix-run/node";
import { useLoaderData } from "@remix-run/react";

export async function loader(_args: LoaderFunctionArgs) {
  const base =
    process.env.FORST_BASE_URL ??
    process.env.FORST_INVOKE_URL ??
    "http://127.0.0.1:8081";

  const [healthRes, versionRes] = await Promise.all([
    fetch(`${base}/health`),
    fetch(`${base}/version`),
  ]);

  return {
    healthy: healthRes.ok,
    version: versionRes.ok ? await versionRes.json() : null,
    base,
  };
}

export default function ApiHealth() {
  const data = useLoaderData<typeof loader>();

  return (
    <main style={{ fontFamily: "monospace", padding: "2rem" }}>
      <h1>Forst health</h1>
      <p>
        Raw HTTP to embedded invoke at <code>{data.base}</code> (same contract as{" "}
        <code>@forst/client</code>).
      </p>
      <pre>{JSON.stringify(data, null, 2)}</pre>
    </main>
  );
}
