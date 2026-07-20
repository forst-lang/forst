/** Minimal module used by integration tests to exercise async export resolution and RPC call paths. */
export async function create(
  amount: number,
  currency: string
): Promise<{ id: string; amount: number }> {
  return { id: "pay_async", amount };
}

/** Concurrent async export so tests can exercise parallel async call handling. */
export async function concurrentEcho(n: number): Promise<{ echo: number }> {
  await new Promise((resolve) => setTimeout(resolve, 5));
  return { echo: n };
}

/** Rejecting async export so tests can verify runtime error propagation from fixture modules. */
export async function failWithError(): Promise<never> {
  throw new Error("Payment provider timeout");
}
