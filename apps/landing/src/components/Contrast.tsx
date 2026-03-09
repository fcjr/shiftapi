const steps = [
  {
    num: "1",
    title: "Define typed handlers",
    desc: (
      <>
        Write standard Go functions with struct input/output types. Use{" "}
        <code>json</code>, <code>query</code>, <code>path</code>, and{" "}
        <code>header</code> struct tags — ShiftAPI routes them automatically.
      </>
    ),
    code: `shiftapi.Post(api, "/greet", greet)`,
  },
  {
    num: "2",
    title: "OpenAPI 3.1 spec generated at runtime",
    desc: (
      <>
        ShiftAPI reflects your Go types into a complete OpenAPI 3.1 schema.
        Served at <code>/openapi.json</code> — always matches your code, never
        maintained by hand.
      </>
    ),
    code: `GET /openapi.json  →  { "openapi": "3.1", ... }`,
  },
  {
    num: "3",
    title: "TypeScript client via HMR",
    desc: (
      <>
        A Vite or Next.js plugin fetches the spec from your running Go server
        and generates a fully-typed client. Save a Go file, types update
        instantly.
      </>
    ),
    code: `const { data } = await client.POST("/greet", ...)
//     ^? { hello: string }`,
  },
];

export function Contrast() {
  return (
    <div className="px-6 pb-[120px]">
      <h2 className="text-center text-[clamp(28px,4vw,40px)] font-extrabold tracking-[-0.03em] mb-4">
        How it works
      </h2>
      <p className="text-center text-[17px] text-text-secondary max-w-[520px] mx-auto mb-14">
        Three layers, one source of truth. No manual steps between them.
      </p>
      <div className="max-w-[720px] mx-auto flex flex-col gap-0">
        {steps.map((step, i) => (
          <div key={step.num} className="relative pl-12 pb-12 max-md:pl-10 max-md:pb-10">
            {/* Vertical line */}
            {i < steps.length - 1 && (
              <div className="absolute left-[15px] top-[36px] bottom-0 w-px bg-border max-md:left-[13px]" />
            )}
            {/* Step number */}
            <div className="absolute left-0 top-0 w-[31px] h-[31px] flex items-center justify-center rounded-full bg-surface border border-border text-[13px] font-bold text-accent-bright font-mono max-md:w-[27px] max-md:h-[27px] max-md:text-[12px]">
              {step.num}
            </div>
            <h3 className="text-[16px] font-bold tracking-[-0.01em] mb-2">
              {step.title}
            </h3>
            <p className="text-[14px] text-text-secondary leading-[1.65] mb-3">
              {step.desc}
            </p>
            <code className="inline-block text-[12.5px] font-mono text-text-muted bg-surface border border-border rounded-lg px-3.5 py-2 whitespace-pre">
              {step.code}
            </code>
          </div>
        ))}
      </div>
    </div>
  );
}
