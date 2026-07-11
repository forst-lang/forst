import type { ActionFunctionArgs, LoaderFunctionArgs } from "@remix-run/node";
import { Form, useLoaderData } from "@remix-run/react";
import {
  AddTodo,
  CompleteTodo,
  GetDashboard,
  ListTodos,
} from "../lib/forst.invoke";

type TodoRow = { id: string; title: string; status: string };

function forstEnv() {
  process.env.FORST_SKIP_SPAWN = "1";
  process.env.FORST_BASE_URL =
    process.env.FORST_BASE_URL ?? "http://127.0.0.1:8081";
}

function parseTodos(encoded: string): TodoRow[] {
  if (!encoded.trim()) return [];
  return encoded.split("\n").map((line) => {
    const [id, title, status] = line.split("\t");
    return { id, title, status };
  });
}

export async function loader(_args: LoaderFunctionArgs) {
  forstEnv();
  const list = await ListTodos();
  const dashboard = await GetDashboard();
  return {
    todos: parseTodos(list.encoded),
    open: list.open,
    done: list.done,
    dashboard,
  };
}

export async function action({ request }: ActionFunctionArgs) {
  forstEnv();
  const form = await request.formData();
  const intent = form.get("intent");

  if (intent === "add") {
    const title = String(form.get("title") ?? "").trim();
    if (title) await AddTodo({ title });
  }

  if (intent === "complete") {
    const id = String(form.get("id") ?? "");
    if (id) await CompleteTodo({ id });
  }

  return null;
}

export default function Index() {
  const data = useLoaderData<typeof loader>();

  return (
    <main style={{ fontFamily: "system-ui, sans-serif", padding: "2rem", maxWidth: 640 }}>
      <h1>Todos</h1>
      <p style={{ color: "#555" }}>
        Remix loader calls Forst on <code>:8081</code>; Forst reads legacy TS on{" "}
        <code>.forst/node.sock</code>.
      </p>

      <section style={{ marginBottom: "1.5rem", padding: "1rem", background: "#f4f4f5", borderRadius: 8 }}>
        <strong>{data.open}</strong> open · <strong>{data.done}</strong> done
        <div style={{ fontSize: 14, marginTop: 8, color: "#444" }}>
          Recent: {data.dashboard.recentTitles || "—"}
        </div>
      </section>

      <Form method="post" style={{ display: "flex", gap: 8, marginBottom: "1.5rem" }}>
        <input type="hidden" name="intent" value="add" />
        <input
          name="title"
          placeholder="Add a todo…"
          required
          style={{ flex: 1, padding: "8px 12px", borderRadius: 6, border: "1px solid #ccc" }}
        />
        <button type="submit" style={{ padding: "8px 16px", borderRadius: 6 }}>
          Add
        </button>
      </Form>

      <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
        {data.todos.map((todo) => (
          <li
            key={todo.id}
            style={{
              display: "flex",
              alignItems: "center",
              gap: 12,
              padding: "10px 0",
              borderBottom: "1px solid #eee",
            }}
          >
            <Form method="post" style={{ margin: 0 }}>
              <input type="hidden" name="intent" value="complete" />
              <input type="hidden" name="id" value={todo.id} />
              <button
                type="submit"
                aria-label={todo.status === "done" ? "Reopen" : "Complete"}
                style={{
                  width: 22,
                  height: 22,
                  borderRadius: 4,
                  border: "2px solid #333",
                  background: todo.status === "done" ? "#333" : "white",
                  cursor: "pointer",
                }}
              />
            </Form>
            <span
              style={{
                textDecoration: todo.status === "done" ? "line-through" : "none",
                color: todo.status === "done" ? "#888" : "inherit",
              }}
            >
              {todo.title}
            </span>
          </li>
        ))}
      </ul>
    </main>
  );
}
