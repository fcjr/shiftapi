import { Fragment } from "react";
import { PipelineArrow } from "../icons";

const steps = [
  { icon: "Go", label: "Structs", desc: "compile time", variant: "go" },
  { icon: "OAS", label: "OpenAPI 3.1", desc: "runtime", variant: "spec" },
  { icon: "TS", label: "Types", desc: "build time", variant: "ts" },
  { icon: "</>", label: "Typed Client", desc: "your frontend", variant: "client" },
] as const;

const variantStyles: Record<string, string> = {
  go: "text-go bg-go/8 border-go/20",
  spec: "text-accent-bright bg-accent/8 border-accent/20",
  ts: "text-ts bg-ts/8 border-ts/20",
  client: "text-green bg-green/8 border-green/20 !text-sm",
};

export function Pipeline() {
  return (
    <div className="px-6 pb-[60px] relative">
      <div className="max-w-[760px] mx-auto flex items-center justify-center gap-5 py-9 px-10 bg-surface border border-border rounded-2xl relative overflow-hidden max-md:flex-col max-md:gap-2 max-md:py-7 max-md:px-6">
        {steps.map((step, i) => (
          <Fragment key={step.variant}>
            {i > 0 && <PipelineArrow />}
            <div className="pipeline-step text-center min-w-[90px]" style={{ "--delay": i } as React.CSSProperties}>
              <div className={`font-mono font-bold text-lg w-11 h-11 flex items-center justify-center mx-auto mb-2 rounded-[10px] border ${variantStyles[step.variant]}`}>
                {step.icon}
              </div>
              <div className="text-[13px] font-semibold mb-0.5">{step.label}</div>
              <div className="text-[11px] text-text-muted uppercase tracking-[0.05em]">{step.desc}</div>
            </div>
          </Fragment>
        ))}
        <div
          className="absolute inset-x-0 bottom-0 h-px opacity-50 max-md:hidden"
          style={{ background: "linear-gradient(90deg, transparent, var(--color-accent), var(--color-go), var(--color-ts), transparent)" }}
        />
      </div>
    </div>
  );
}
