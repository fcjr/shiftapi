"use client";

import { useState } from "react";
import { client } from "@shiftapi/client";

export default function Home() {
  const [name, setName] = useState("");
  const [result, setResult] = useState("");

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;

    setResult("Loading...");
    const { data, error } = await client.POST("/echo", {
      body: { message: name.trim() },
    });

    if (error) {
      setResult(`Error: ${error.message}`);
    } else {
      setResult(`Echo: ${data.message}`);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center">
      <main className="flex flex-col items-center gap-8">
        <h1 className="text-3xl font-semibold">{"{{name}}"}</h1>
        <form onSubmit={handleSubmit} className="flex gap-2">
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Enter a message"
            className="rounded border px-3 py-2 dark:bg-zinc-800 dark:border-zinc-700"
          />
          <button
            type="submit"
            className="rounded bg-foreground px-4 py-2 text-background"
          >
            Send
          </button>
        </form>
        {result && <pre className="text-lg">{result}</pre>}
      </main>
    </div>
  );
}
