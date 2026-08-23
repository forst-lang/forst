import { capturePayment } from "../services/payment";
import { quoteTotal } from "../services/pricing";

export function createOrder(
  amount: number,
  currency: string
): { id: string; total: string; totalCents: number } {
  const quote = quoteTotal(amount, currency);
  const id = capturePayment(amount, currency);
  return { id, total: quote.formatted, totalCents: quote.cents };
}
