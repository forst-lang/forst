/** Minimal module used by integration tests to exercise arrow-function export resolution. */
export const create = (amount: number, currency: string): { id: string } => {
  return { id: `${amount}-${currency}` };
};
