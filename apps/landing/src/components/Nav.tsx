import { useCountUp, useStarCount } from "../hooks";
import { LogoIcon, StarIcon } from "../icons";

export function Nav() {
  const count = useStarCount();
  const display = useCountUp(count);

  return (
    <header className="wrap flex items-center justify-between h-[72px]">
      <a href="/" className="flex items-center gap-2.5 font-bold text-[18px] tracking-[-0.02em]">
        <LogoIcon size={26} />
        <span>ShiftAPI</span>
      </a>
      <nav className="flex items-center gap-7 max-md:gap-5 text-[15px] font-medium text-chalk-2">
        <a href="/docs/getting-started/introduction" className="hover:text-chalk transition-colors">Docs</a>
        <a href="https://pkg.go.dev/github.com/fcjr/shiftapi" target="_blank" rel="noopener" className="hover:text-chalk transition-colors max-md:hidden">Go reference</a>
        <a href="https://github.com/fcjr/shiftapi" target="_blank" rel="noopener" className="inline-flex items-center gap-1.5 hover:text-chalk transition-colors">
          GitHub
          {count !== null && (
            <span className="inline-flex items-center gap-1 text-chalk-3 text-[13px] tabular-nums">
              <StarIcon width={12} height={12} />
              {display}
            </span>
          )}
        </a>
      </nav>
    </header>
  );
}
