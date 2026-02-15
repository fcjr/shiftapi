import { client } from "@shiftapi/client";

// ---- Type-safe API calls ----

// GET /health — response is typed as { ok?: boolean }
async function checkHealth() {
  const { data, error } = await client.GET("/health");
  if (error) {
    console.error("Health check failed:", error.message);
    return;
  }
  console.log("Health:", data);
}

// POST /greet — body is typed as { name: string (required) }, response as { hello?: string }
async function greet(name: string) {
  const { data, error } = await client.POST("/greet", {
    body: { name },
  });
  if (error) {
    return `Error: ${error.message}`;
  }
  return `Hello response: ${data.hello}`;
}

// ---- Wire up the UI ----

const output = document.getElementById("output")!;
const form = document.getElementById("greet-form")!;
const input = document.getElementById("name-input") as HTMLInputElement;

checkHealth().then(() => {
  output.textContent = "Health check passed. Try greeting someone.";
});

form.addEventListener("submit", async (e) => {
  e.preventDefault();
  const name = input.value.trim();
  if (!name) return;
  output.textContent = "Loading...";
  output.textContent = await greet(name);
});
