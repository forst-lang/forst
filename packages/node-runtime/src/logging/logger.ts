/** Structured logs on stderr only — stdout is reserved for RPC. */
export function log(
  event: string,
  fields: Record<string, string | number | boolean | null | undefined> = {}
): void {
  const payload = {
    component: "node-runtime",
    event,
    ...fields,
    ts: Date.now(),
  };
  process.stderr.write(`${JSON.stringify(payload)}\n`);
}
