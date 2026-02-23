import { useState } from "react";
import { api } from "@{{name}}/api";

export default function App() {
  const [message, setMessage] = useState("");
  const health = api.useQuery("get", "/health");
  const echo = api.useMutation("post", "/echo");

  if (health.isLoading) return <p>Loading...</p>;
  if (health.error) return <p>Health check failed: {health.error.message}</p>;

  return (
    <div>
      <h1>{{name}}</h1>
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
      >
        <input
          type="text"
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          placeholder="Enter a message"
        />
        <button type="submit" disabled={echo.isPending}>
          Send
        </button>
      </form>
      {echo.isPending && <p>Loading...</p>}
      {echo.error && <p>Error: {echo.error.message}</p>}
      {echo.data && <pre>Echo: {echo.data.message}</pre>}
    </div>
  );
}
