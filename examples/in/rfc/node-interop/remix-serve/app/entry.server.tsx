import type { EntryContext } from "@remix-run/node";
import { RemixServer } from "@remix-run/react";
import { renderToString } from "react-dom/server";

declare global {
  // eslint-disable-next-line no-var
  var __forstTodos:
    | { nextId: number; editCount: number; items: { id: string; title: string; status: string }[] }
    | undefined;
}

export default function handleRequest(
  request: Request,
  responseStatusCode: number,
  responseHeaders: Headers,
  remixContext: EntryContext
) {
  if (!globalThis.__forstTodos) {
    globalThis.__forstTodos = {
      nextId: 2,
      editCount: 1,
      items: [
        { id: "1", title: "Migrate checkout to Forst", status: "open" },
      ],
    };
  }

  const markup = renderToString(
    <RemixServer context={remixContext} url={request.url} />
  );

  responseHeaders.set("Content-Type", "text/html");

  return new Response("<!DOCTYPE html>" + markup, {
    status: responseStatusCode,
    headers: responseHeaders,
  });
}
