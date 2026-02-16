<script lang="ts">
  import { client } from "@shiftapi/client";

  let output = $state("");

  $effect(() => {
    client.GET("/health").then(({ error }) => {
      if (error) {
        output = `Health check failed: ${error.message}`;
      } else {
        output = "Health check passed. Try sending a message.";
      }
    });
  });

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    const formData = new FormData(e.currentTarget as HTMLFormElement);
    const message = (formData.get("message") as string).trim();
    if (!message) return;
    output = "Loading...";
    const { data, error } = await client.POST("/echo", { body: { message } });
    if (error) {
      output = `Error: ${error.message}`;
    } else {
      output = `Echo: ${data.message}`;
    }
  }
</script>

<div>
  <h1>{{name}}</h1>
  <form onsubmit={handleSubmit}>
    <input type="text" name="message" placeholder="Enter a message" />
    <button type="submit">Send</button>
  </form>
  <pre>{output}</pre>
</div>
