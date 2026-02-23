<script lang="ts">
  import { $api } from "./api";

  let message = $state("");
  const health = $api.createQuery("get", "/health");
  const echo = $api.createMutation("post", "/echo");
</script>

<div>
  <h1>{{name}}</h1>
  {#if health.isLoading}
    <p>Loading...</p>
  {:else if health.error}
    <p>Health check failed: {health.error.message}</p>
  {:else}
    <input
      type="text"
      bind:value={message}
      placeholder="Enter a message"
    />
    <button onclick={() => echo.mutate({ body: { message } })}>Send</button>
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
