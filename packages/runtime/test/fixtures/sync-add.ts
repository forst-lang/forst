/** Minimal module used by integration tests to exercise synchronous export resolution and RPC calls. */
export function add(a: number, b: number): number {
  return a + b;
}

/** Named string export so tests can verify multiple sync exports from one fixture module. */
export function greet(name: string): string {
  return `hello ${name}`;
}
