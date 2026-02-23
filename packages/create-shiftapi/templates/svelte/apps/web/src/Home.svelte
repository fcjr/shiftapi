<script lang="ts">
  import { api } from "@{{name}}/api";

  let message = $state("");
  const health = api.createQuery("get", "/health");
  const echo = api.createMutation("post", "/echo");

  function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    const trimmed = message.trim();
    if (!trimmed) return;
    echo.mutate(
      { body: { message: trimmed } },
      { onSuccess: () => { message = ""; } },
    );
  }
</script>

<div>
  <h1>{{name}}</h1>
  {#if health.isLoading}
    <p>Loading...</p>
  {:else if health.error}
    <p>Health check failed: {health.error.message}</p>
  {:else}
    <form onsubmit={handleSubmit}>
      <input
        type="text"
        bind:value={message}
        placeholder="Enter a message"
      />
      <button type="submit" disabled={echo.isPending}>Send</button>
    </form>
    {#if echo.isPending}
      <p>Loading...</p>
    {/if}
    {#if echo.error}
      <p>Error: {echo.error.message}</p>
    {/if}
    {#if echo.data}
      <pre>Echo: {echo.data.message}</pre>
    {/if}
  {/if}
</div>
