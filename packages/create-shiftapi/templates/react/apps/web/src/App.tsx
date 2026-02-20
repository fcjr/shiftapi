import { useState } from "react";
import { $api } from "./api";

export default function App() {
  const [message, setMessage] = useState("");
  const health = $api.useQuery("get", "/health");
  const echo = $api.useMutation("post", "/echo");

  if (health.isLoading) return <p>Loading...</p>;
  if (health.error) return <p>Health check failed: {health.error.message}</p>;

  return (
    <div>
      <h1>{{name}}</h1>
      <input
        type="text"
        value={message}
        onChange={(e) => setMessage(e.target.value)}
        placeholder="Enter a message"
      />
      <button onClick={() => echo.mutate({ body: { message } })}>Send</button>
      {echo.isPending && <p>Loading...</p>}
      {echo.error && <p>Error: {echo.error.message}</p>}
      {echo.data && <pre>Echo: {echo.data.message}</pre>}
    </div>
  );
}
