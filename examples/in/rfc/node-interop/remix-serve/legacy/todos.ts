type Todo = { id: string; title: string; status: "open" | "done" };

declare global {
  // eslint-disable-next-line no-var
  var __forstTodos:
    | { nextId: number; editCount: number; items: Todo[] }
    | undefined;
}

function store() {
  if (!globalThis.__forstTodos) {
    globalThis.__forstTodos = {
      nextId: 2,
      editCount: 1,
      items: [
        { id: "1", title: "Migrate checkout to Forst", status: "open" },
      ],
    };
  }
  return globalThis.__forstTodos;
}

/** Sync counter shared with Remix — used by host-mode integration tests. */
export function bumpEditCount(): number {
  console.error("[legacy/todos] bumpEditCount");
  return ++store().editCount;
}

export function openCount(): number {
  console.error("[legacy/todos] openCount");
  return store().items.filter((t) => t.status === "open").length;
}

export function todoCount(): number {
  console.error("[legacy/todos] todoCount");
  return store().items.length;
}

/** Tab-separated rows: id, title, status — one line per todo. */
export function formatTodoList(): string {
  console.error("[legacy/todos] formatTodoList");
  return store()
    .items.map((t) => t.id + "\t" + t.title + "\t" + t.status)
    .join("\n");
}

export function addTodo(title: string) {
  console.error("[legacy/todos] addTodo", { title });
  const s = store();
  const id = String(s.nextId++);
  s.items.push({ id, title, status: "open" });
  s.editCount += 1;
  return { id, title, status: "open" };
}

export function toggleTodo(id: string) {
  console.error("[legacy/todos] toggleTodo", { id });
  const s = store();
  const todo = s.items.find((t) => t.id === id);
  if (!todo) {
    return { id, title: "", status: "open" };
  }
  todo.status = todo.status === "open" ? "done" : "open";
  s.editCount += 1;
  return { id: todo.id, title: todo.title, status: todo.status };
}

export function* allTodos(): Generator<Todo> {
  for (const todo of store().items) {
    yield todo;
  }
}

/** Simulates async persistence to a legacy notification service. */
export async function persistSnapshot(): Promise<{ savedAt: string }> {
  console.error("[legacy/todos] persistSnapshot");
  await new Promise((resolve) => setTimeout(resolve, 1));
  return { savedAt: "ok" };
}
