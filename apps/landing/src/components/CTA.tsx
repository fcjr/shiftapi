import { useState } from "react";
import { ArrowRightIcon, CheckIcon, CopyIcon } from "../icons";

export function CTA() {
  const [copied, setCopied] = useState(false);

  const copy = () => {
    navigator.clipboard.writeText("npm create shiftapi@latest");
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="px-6 pb-[120px]">
      <div
        className="max-w-[680px] mx-auto text-center py-16 px-12 rounded-2xl border border-border relative overflow-hidden max-md:py-12 max-md:px-6"
        style={{ background: "radial-gradient(ellipse 60% 40% at 50% 0%, rgba(196,240,66,0.04), transparent), radial-gradient(ellipse 60% 40% at 50% 100%, rgba(0,173,216,0.04), transparent), var(--color-surface)" }}
      >
        <div
          className="absolute top-0 inset-x-0 h-px opacity-40"
          style={{ background: "linear-gradient(90deg, transparent, var(--color-accent), var(--color-go), var(--color-ts), transparent)" }}
        />
        <h2 className="text-[clamp(24px,4vw,36px)] font-extrabold tracking-[-0.03em] mb-3">
          Start building in 30 seconds
        </h2>
        <p className="text-base text-text-secondary mb-8 max-w-[440px] mx-auto">
          One command scaffolds a full-stack Go + React, Svelte, or Next.js app with end-to-end types.
        </p>
        <div className="inline-flex items-center gap-2.5 py-3.5 pl-6 pr-3 bg-elevated border border-border rounded-[10px] text-[15px] mb-8">
          <span className="text-accent-bright font-mono font-medium select-none">$</span>
          <code className="text-text-secondary">npm create shiftapi@latest</code>
          <button
            className="flex items-center justify-center text-text-muted cursor-pointer p-1.5 rounded-md transition-all duration-150 hover:text-text hover:bg-white/5"
            aria-label="Copy to clipboard"
            onClick={copy}
          >
            {copied ? <CheckIcon className="text-green" /> : <CopyIcon />}
          </button>
        </div>
        <div className="flex justify-center gap-3 flex-wrap">
          <a
            href="https://github.com/fcjr/shiftapi"
            className="inline-flex items-center gap-2 px-6 py-3 rounded-[10px] text-sm font-semibold bg-accent text-bg transition-all duration-200 hover:bg-accent-bright hover:-translate-y-px hover:shadow-[0_4px_20px_rgba(196,240,66,0.2)]"
            target="_blank"
            rel="noopener"
          >
            View on GitHub <ArrowRightIcon />
          </a>
          <a
            href="/docs/getting-started/introduction"
            className="inline-flex items-center gap-2 px-6 py-3 rounded-[10px] text-sm font-semibold bg-transparent text-text-secondary border border-border transition-all duration-200 hover:text-text hover:border-border-hover hover:bg-elevated"
          >
            Read the Docs
          </a>
        </div>
      </div>
    </div>
  );
}
