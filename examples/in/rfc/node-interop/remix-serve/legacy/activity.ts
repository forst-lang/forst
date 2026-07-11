export function* recentTitles(limit: number): Generator<string> {
  console.error("[legacy/activity] recentTitles", { limit });
  const todos = globalThis.__forstTodos?.items ?? [];
  let n = 0;
  for (const todo of todos) {
    if (n >= limit) break;
    yield todo.title;
    n += 1;
  }
}

export async function* activityFeed(
  userId: string
): AsyncGenerator<{ kind: string }> {
  console.error("[legacy/activity] activityFeed", { userId });
  yield { kind: "feed-open:" + userId };
  yield { kind: "feed-tick:" + userId };
}

export async function dispatchActivity(evt: {
  kind: string;
}): Promise<void> {
  console.error("[legacy/activity] dispatchActivity", evt);
  await new Promise((resolve) => setTimeout(resolve, 1));
}
