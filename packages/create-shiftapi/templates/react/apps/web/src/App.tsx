import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { client } from "@shiftapi/client";

export default function App() {
  const [output, setOutput] = useState("");

  useEffect(() => {
    client.GET("/health").then(({ error }) => {
      if (error) {
        setOutput(`Health check failed: ${error.message}`);
      } else {
        setOutput("Health check passed. Try sending a message.");
      }
    });
  }, []);

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const formData = new FormData(e.currentTarget);
    const message = (formData.get("message") as string).trim();
    if (!message) return;
    setOutput("Loading...");
    const { data, error } = await client.POST("/echo", { body: { message } });
    if (error) {
      setOutput(`Error: ${error.message}`);
    } else {
      setOutput(`Echo: ${data.message}`);
    }
  }

  return (
    <div>
      <h1>{{name}}</h1>
      <form onSubmit={handleSubmit}>
        <input type="text" name="message" placeholder="Enter a message" />
        <button type="submit">Send</button>
      </form>
      <pre>{output}</pre>
    </div>
  );
}
