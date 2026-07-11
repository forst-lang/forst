export function toCents(amount: number): number {
  return Math.round(amount * 100);
}

export function formatMoney(cents: number, currency: string): string {
  return `${currency}:${(cents / 100).toFixed(2)}`;
}
