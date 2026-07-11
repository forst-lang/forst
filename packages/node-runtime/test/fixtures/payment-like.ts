export interface PaymentResult {
  id: string;
}

export interface PaymentEvent {
  type: string;
}

export async function create(
  amount: number,
  currency: string,
): Promise<PaymentResult> {
  return { id: `${amount}-${currency}` };
}

export async function* watchEvents(
  accountId: string,
): AsyncGenerator<PaymentEvent> {
  yield { type: accountId };
}

export function* readChunks(path: string): Generator<Buffer> {
  yield Buffer.from(path);
}

export function formatAmount(amount: number): string {
  return amount.toFixed(2);
}
