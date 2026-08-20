const features = [
  {
    icon: <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>,
    title: "Type-safe handlers",
    desc: <>Generic Go functions capture request and response types at compile time. <code>json</code>, <code>query</code>, <code>header</code>, and <code>form</code> tags decide where each field is read from.</>,
  },
  {
    icon: <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>,
    title: "Validation included",
    desc: <>ShiftAPI enforces struct tags like <code>validate:"required,email"</code> on every request and copies the same rules into your OpenAPI schema.</>,
  },
  {
    icon: <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/></svg>,
    title: "Vite & Next.js",
    desc: "Plugins that start your Go server, proxy API requests to it, and regenerate types on every save.",
  },
  {
    icon: <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>,
    title: "File uploads",
    desc: <>Declare uploads with <code>form</code> tags. The TypeScript client gets the matching <code>multipart/form-data</code> types.</>,
  },
  {
    icon: <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="2" y="3" width="20" height="14" rx="2" ry="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>,
    title: "Interactive docs",
    desc: <>A Scalar API reference at <code>/docs</code> and the raw spec at <code>/openapi.json</code>. Both come from the same generated schema.</>,
  },
  {
    icon: <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M4 12h8"/><path d="M4 18V6"/><path d="M12 18V6"/><path d="M16 6l4 6-4 6"/></svg>,
    title: "Real-time",
    desc: <><code>HandleSSE</code> gives you typed server push through an <code>sse</code> helper. <code>HandleWS</code> does the same for bidirectional WebSockets.</>,
  },
];

export function Features() {
  return (
    <section className="px-6 pb-[120px]">
      <h2 className="text-center text-[clamp(28px,4vw,40px)] font-extrabold tracking-[-0.03em] mb-14">
        The parts you would otherwise wire up yourself
      </h2>
      <div className="max-w-[960px] mx-auto grid grid-cols-3 gap-4 max-md:grid-cols-1">
        {features.map((f) => (
          <div
            key={f.title}
            className="p-7 bg-surface border border-border rounded-2xl transition-[border-color,transform] duration-250 hover:border-border-hover hover:-translate-y-0.5"
          >
            <div className="w-[42px] h-[42px] flex items-center justify-center bg-accent/7 border border-accent/12 rounded-[10px] text-accent-bright mb-[18px]">
              {f.icon}
            </div>
            <h3 className="text-[15px] font-bold mb-2 tracking-[-0.01em]">{f.title}</h3>
            <p className="text-[13.5px] text-text-secondary leading-[1.65]">{f.desc}</p>
          </div>
        ))}
      </div>
    </section>
  );
}
