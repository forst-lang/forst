export const create = (amount: number, currency: string): { id: string } => {
  return { id: `${amount}-${currency}` };
};
