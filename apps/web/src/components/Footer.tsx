import { LogoIcon } from "../icons";

export function Footer() {
  return (
    <footer className="border-t border-border py-7 px-6">
      <div className="max-w-[1100px] mx-auto flex items-center justify-between max-md:flex-col max-md:gap-4 max-md:text-center">
        <div className="flex items-center gap-2 font-semibold text-[13px] text-text-muted">
          <LogoIcon size={18} />
          <span>ShiftAPI</span>
        </div>
        <div className="flex gap-6">
          <a href="https://github.com/fcjr/shiftapi" target="_blank" rel="noopener" className="text-[13px] text-text-muted transition-colors hover:text-text">GitHub</a>
          <a href="https://pkg.go.dev/github.com/fcjr/shiftapi" target="_blank" rel="noopener" className="text-[13px] text-text-muted transition-colors hover:text-text">Go Reference</a>
          <a href="https://www.npmjs.com/package/shiftapi" target="_blank" rel="noopener" className="text-[13px] text-text-muted transition-colors hover:text-text">npm</a>
        </div>
      </div>
    </footer>
  );
}
