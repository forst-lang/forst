/** Minimal module used by integration tests to exercise export resolution for payment-like APIs. */
export interface PaymentResult {
  id: string;
}

/** Minimal event shape for generator export indexing in payment-like fixture tests. */
export interface PaymentEvent {
  type: string;
}

/** Async factory export so tests can index and invoke promise-returning module exports. */
export async function create(
  amount: number,
  currency: string,
): Promise<PaymentResult> {
  return { id: `${amount}-${currency}` };
}

/** Async generator export so tests can exercise streaming export discovery and calls. */
export async function* watchEvents(
  accountId: string,
): AsyncGenerator<PaymentEvent> {
  yield { type: accountId };
}

/** Sync generator export so tests can distinguish generator kinds during indexing. */
export function* readChunks(path: string): Generator<Buffer> {
  yield Buffer.from(path);
}

/** Simple sync helper export for mixed export-kind coverage in the same fixture module. */
export function formatAmount(amount: number): string {
  return amount.toFixed(2);
}
