import { client } from "@shiftapi/client";
import { GreetForm } from "./greet-form";

export default async function Home() {
  const { data, error } = await client.GET("/health");

  return (
    <main style={{ fontFamily: "system-ui", padding: "2rem" }}>
      <h1>ShiftAPI + Next.js Example</h1>
      <p>
        Health check:{" "}
        {error ? (
          <span style={{ color: "red" }}>Failed: {error.message}</span>
        ) : (
          <span style={{ color: "green" }}>OK ({JSON.stringify(data)})</span>
        )}
      </p>
      <hr />
      <GreetForm />
    </main>
  );
}
