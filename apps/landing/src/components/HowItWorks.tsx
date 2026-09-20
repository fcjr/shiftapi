const steps = [
  ["When you compile", "Registering a handler captures its input and output structs. Field tags say where each value is read from."],
  ["When the server starts", "Those types become an OpenAPI 3.1 document served by the same process, so the spec cannot drift from the code."],
  ["When you save", "The Vite or Next.js plugin fetches the spec and rewrites the client. Types update in your editor without a reload."],
];

export function HowItWorks() {
  return (
    <section className="wrap pt-6 pb-16 max-md:pb-12">
      <ol className="grid grid-cols-3 gap-x-10 gap-y-8 max-md:grid-cols-1">
        {steps.map(([when, text]) => (
          <li key={when}>
            <h2 className="h3">{when}</h2>
            <p className="text-chalk-2 mt-2 text-[16px] leading-[1.55] max-w-[30em]">{text}</p>
          </li>
        ))}
      </ol>
      <p className="prose-inline text-chalk-2 mt-14 max-w-[42em] text-[16px] leading-[1.6]">
        Validation tags, typed errors, file uploads, server-sent events, WebSockets, and a Scalar
        reference at <code>/docs</code> come along with it. The API is still an{" "}
        <code>http.Handler</code>, so your middleware and deployment stay the same.{" "}
        <a href="/docs/getting-started/introduction" className="link">Read the docs</a> for the rest.
      </p>
    </section>
  );
}
