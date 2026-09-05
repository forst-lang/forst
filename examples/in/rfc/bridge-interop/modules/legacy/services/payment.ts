import { formatMoney, toCents } from "../utils/money";
import { ID_PREFIX } from "./constants";

let seq = 0;

export function capturePayment(amount: number, currency: string): string {
  seq += 1;
  const cents = toCents(amount);
  return ID_PREFIX + String(seq) + ":" + formatMoney(cents, currency);
}
