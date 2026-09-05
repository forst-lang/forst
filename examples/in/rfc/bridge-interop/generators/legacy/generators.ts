export function* syncNumbers(limit: number): Generator<number> {
  for (let i = 0; i < limit; i++) {
    yield i;
  }
}

export async function* asyncNumbers(limit: number): AsyncGenerator<number> {
  for (let i = 0; i < limit; i++) {
    yield i;
  }
}

export function* emptyGen(): Generator<number> {}

export function* throwGen(): Generator<number> {
  throw new Error("generator failed");
}

export function* returnGen(): Generator<string> {
  yield "a";
  return "done-value";
}

let finallyCount = 0;

export function finallyRunCount(): number {
  return finallyCount;
}

export function resetFinallyCount(): void {
  finallyCount = 0;
}

export function* withFinally(): Generator<number> {
  try {
    yield 1;
    yield 2;
    yield 3;
  } finally {
    finallyCount += 1;
  }
}
