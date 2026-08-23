export interface Event {
  type: string;
}

export async function* subscribe(userId: string): AsyncGenerator<Event> {
  yield { type: "open:" + userId };
  yield { type: "tick:" + userId };
}

export async function dispatch(evt: Event): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve, 1));
}
