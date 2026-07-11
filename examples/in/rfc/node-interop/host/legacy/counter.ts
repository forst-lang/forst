declare const globalThis: { __forstTest?: { n: number } };

export function inc(): number {
  if (!globalThis.__forstTest) {
    globalThis.__forstTest = { n: 0 };
  }
  return ++globalThis.__forstTest.n;
}
