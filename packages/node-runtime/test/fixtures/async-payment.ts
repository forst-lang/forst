export async function create(
  amount: number,
  currency: string
): Promise<{ id: string; amount: number }> {
  return { id: "pay_async", amount };
}

export async function concurrentEcho(n: number): Promise<{ echo: number }> {
  await new Promise((resolve) => setTimeout(resolve, 5));
  return { echo: n };
}

export async function failWithError(): Promise<never> {
  throw new Error("Payment provider timeout");
}
