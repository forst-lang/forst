import { TAX_RATE } from "./constants";

export function withTax(cents: number): number {
  return Math.round(cents * (1 + TAX_RATE));
}
