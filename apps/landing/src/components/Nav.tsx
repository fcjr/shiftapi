import { useCountUp, useStarCount } from "../hooks";
import { GitHubIcon, LogoIcon, StarIcon } from "../icons";

export function Nav() {
  const count = useStarCount();
  const display = useCountUp(count);

  return (
    <nav className="fixed top-0 inset-x-0 z-100 bg-bg/80 backdrop-blur-[20px] backdrop-saturate-[1.4] border-b border-white/[0.04]">
      <div className="max-w-[1100px] mx-auto px-6 h-[60px] flex items-center justify-between">
        <a href="/" className="flex items-center gap-2.5 font-bold text-[17px] tracking-[-0.02em]">
          <LogoIcon size={28} />
          <span>ShiftAPI</span>
        </a>
        <div className="flex items-center gap-6 max-md:gap-3.5">
          <a href="/docs/getting-started/introduction" className="text-text-secondary text-sm font-medium transition-colors hover:text-text">Docs</a>
          <a href="https://github.com/fcjr/shiftapi" target="_blank" rel="noopener" className="text-text-secondary text-sm font-medium transition-colors hover:text-text">GitHub</a>
          <a
            href="https://github.com/fcjr/shiftapi"
            className="group hidden md:inline-flex items-center rounded-lg border border-border bg-elevated text-[13px] font-semibold text-text-secondary overflow-hidden transition-[border-color,background] duration-200 hover:border-border-hover hover:bg-surface"
            target="_blank"
            rel="noopener"
          >
            <GitHubIcon className="py-1.5 pl-2.5 opacity-60 transition-opacity group-hover:opacity-100" />
            <span className="inline-flex items-center gap-1 py-1.5 px-2 text-text-secondary transition-colors group-hover:text-text">
              <StarIcon className="fill-none transition-[fill,stroke,transform] duration-250 ease-[cubic-bezier(0.34,1.56,0.64,1)] group-hover:fill-accent group-hover:stroke-accent group-hover:scale-115" />
              Star
            </span>
            {display && (
              <span className="text-xs font-semibold py-1.5 px-2.5 border-l border-border text-text-muted bg-white/[0.02] transition-colors group-hover:text-text-secondary">
                {display}
              </span>
            )}
          </a>
        </div>
      </div>
    </nav>
  );
}
