import { useState } from "react";
import { ArrowRightIcon, CheckIcon, CopyIcon } from "../icons";

export function Hero() {
  const [copied, setCopied] = useState(false);

  const copy = () => {
    navigator.clipboard.writeText("npm create shiftapi@latest");
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <section className="max-w-[800px] mx-auto px-6 pt-[140px] pb-[100px] text-center max-md:pt-[110px] max-md:pb-20 max-md:px-5">
      <p className="inline-block text-[13px] font-mono font-medium text-text-muted mb-6 px-3 py-1.5 bg-surface border border-border rounded-full">
        A Go framework for end-to-end type-safe APIs
      </p>
      <h1 className="text-[clamp(40px,7vw,72px)] font-black leading-[1.05] tracking-[-0.04em] mb-7">
        Write <span className="text-go">Go</span>.<br /><span className="text-ts">TypeScript</span> keeps up.
      </h1>
      <p className="text-[19px] text-text-secondary max-w-[600px] mx-auto mb-11 leading-[1.7]">
        Define typed handler functions in Go. ShiftAPI generates an OpenAPI 3.1 spec
        at runtime and a fully-typed TypeScript client via Vite/Next.js HMR — no
        codegen CLI, no manual spec, no drift.
      </p>
      <div className="flex items-center justify-center gap-3.5 flex-wrap max-md:flex-col">
        <div className="flex items-center gap-2.5 py-2.5 pl-[18px] pr-3 bg-surface border border-border rounded-[10px] text-sm transition-[border-color] duration-200 hover:border-border-hover">
          <span className="text-accent-bright font-mono font-medium select-none">$</span>
          <code className="text-text-secondary whitespace-nowrap">npm create shiftapi@latest</code>
          <button
            className="flex items-center justify-center text-text-muted cursor-pointer p-1.5 rounded-md transition-all duration-150 hover:text-text hover:bg-white/5"
            aria-label="Copy to clipboard"
            onClick={copy}
          >
            {copied ? <CheckIcon className="text-green" /> : <CopyIcon />}
          </button>
        </div>
        <a
          href="https://github.com/fcjr/shiftapi"
          className="inline-flex items-center gap-2 px-6 py-3 rounded-[10px] text-sm font-semibold bg-accent text-bg transition-all duration-200 hover:bg-accent-bright hover:-translate-y-px hover:shadow-[0_4px_20px_rgba(196,240,66,0.2)]"
          target="_blank"
          rel="noopener"
        >
          Get Started <ArrowRightIcon />
        </a>
      </div>
    </section>
  );
}
