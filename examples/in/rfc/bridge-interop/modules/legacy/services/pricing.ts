import { withTax } from "./tax";
import { formatMoney } from "../utils/money";

export function quoteTotal(
  amount: number,
  currency: string
): { formatted: string; cents: number } {
  const base = Math.round(amount * 100);
  const taxed = withTax(base);
  return { cents: taxed, formatted: formatMoney(taxed, currency) };
}
