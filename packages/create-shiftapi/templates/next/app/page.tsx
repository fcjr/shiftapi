"use client";

import { useState } from "react";
import { api } from "@/api";

export default function Home() {
  const [message, setMessage] = useState("");
  const health = api.useQuery("get", "/health");
  const echo = api.useMutation("post", "/echo");

  if (health.isLoading) return <p>Loading...</p>;
  if (health.error) return <p>Health check failed: {health.error.message}</p>;

  return (
    <div className="flex min-h-screen items-center justify-center">
      <main className="flex flex-col items-center gap-8">
        <h1 className="text-3xl font-semibold">{"{{name}}"}</h1>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            const trimmed = message.trim();
            if (!trimmed) return;
            echo.mutate(
              { body: { message: trimmed } },
              { onSuccess: () => setMessage("") },
            );
          }}
          className="flex gap-2"
        >
          <input
            type="text"
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            placeholder="Enter a message"
            className="rounded border px-3 py-2 dark:bg-zinc-800 dark:border-zinc-700"
          />
          <button
            type="submit"
            disabled={echo.isPending}
            className="rounded bg-foreground px-4 py-2 text-background"
          >
            Send
          </button>
        </form>
        {echo.isPending && <p>Loading...</p>}
        {echo.error && <p>Error: {echo.error.message}</p>}
        {echo.data && <pre className="text-lg">Echo: {echo.data.message}</pre>}
      </main>
    </div>
  );
}
