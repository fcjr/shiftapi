"use client";

import { useState } from "react";
import { client } from "@shiftapi/client";

export function GreetForm() {
  const [name, setName] = useState("");
  const [result, setResult] = useState("");

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;

    setResult("Loading...");
    const { data, error } = await client.POST("/greet", {
      body: { name: name.trim() },
    });

    if (error) {
      setResult(`Error: ${error.message}`);
    } else {
      setResult(`Hello response: ${data.hello}`);
    }
  }

  return (
    <div>
      <h2>Greet</h2>
      <form onSubmit={handleSubmit}>
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Enter a name"
        />
        <button type="submit">Greet</button>
      </form>
      {result && <pre>{result}</pre>}
    </div>
  );
}
