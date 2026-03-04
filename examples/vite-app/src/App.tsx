import { useState, useRef } from "react";
import { api } from "./api";

export default function App() {
  const [name, setName] = useState("");
  const fileRef = useRef<HTMLInputElement>(null);
  const health = api.useQuery("get", "/health");
  const greet = api.useMutation("post", "/greet");
  const upload = api.useMutation("post", "/upload");

  if (health.isLoading) return <p>Loading...</p>;
  if (health.error) return <p>Health check failed: {health.error.message}</p>;

  return (
    <div>
      <h1>ShiftAPI + Vite Example</h1>

      <h2>Greet</h2>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          const trimmed = name.trim();
          if (!trimmed) return;
          greet.mutate(
            { body: { name: trimmed } },
            { onSuccess: () => setName("") },
          );
        }}
      >
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Enter a name"
        />
        <button type="submit" disabled={greet.isPending}>
          Greet
        </button>
      </form>
      {greet.isPending && <p>Loading...</p>}
      {greet.error && <p>Error: {greet.error.message}</p>}
      {greet.data && <pre>Hello: {greet.data.hello}</pre>}

      <h2>Upload</h2>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          const file = fileRef.current?.files?.[0];
          if (!file) return;
          upload.mutate(
            { body: { file } },
            {
              onSuccess: () => {
                if (fileRef.current) fileRef.current.value = "";
              },
            },
          );
        }}
      >
        <input ref={fileRef} type="file" />
        <button type="submit" disabled={upload.isPending}>
          Upload
        </button>
      </form>
      {upload.isPending && <p>Uploading...</p>}
      {upload.error && <p>Error: {upload.error.message}</p>}
      {upload.data && (
        <pre>
          Uploaded: {upload.data.filename} ({upload.data.size} bytes)
        </pre>
      )}
    </div>
  );
}
