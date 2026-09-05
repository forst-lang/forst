export async function create(
  amount: number,
  currency: string
): Promise<{ id: string; amount: number; currency: string }> {
  return { id: "pay_1", amount, currency };
}

export async function failWithError(): Promise<{ id: string }> {
  throw new Error("Payment provider timeout");
}

export async function failWithObject(): Promise<{ id: string }> {
  return Promise.reject({ code: "DECLINED", reason: "insufficient funds" });
}

export async function delayed(ms: number): Promise<{ waited: number }> {
  await new Promise((resolve) => setTimeout(resolve, ms));
  return { waited: ms };
}

export async function concurrentEcho(n: number): Promise<{ echo: number }> {
  await new Promise((resolve) => setTimeout(resolve, 5));
  return { echo: n };
}
