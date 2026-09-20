import { Install } from "./Install";
import { LiveDemo } from "./LiveDemo";

export function Hero() {
  return (
    <section className="wrap pt-8 pb-16 max-md:pt-6 max-md:pb-12">
      <div className="grid grid-cols-[1.35fr_1fr] gap-x-14 gap-y-7 items-end max-md:grid-cols-1">
        <h1 className="display text-[clamp(40px,5.4vw,66px)]">
          Write Go.
          <br />
          TypeScript keeps up.
        </h1>
        <div className="pb-1">
          <p className="lead text-[17px] leading-[1.5]">
            A Go framework that turns your handler types into an OpenAPI 3.1 spec at runtime and
            a typed TypeScript client on every save.
          </p>
          <div className="flex flex-wrap items-center gap-x-5 gap-y-3 mt-5">
            <Install />
            <a href="/docs/getting-started/quickstart" className="link text-[15px]">
              or read the quickstart
            </a>
          </div>
        </div>
      </div>
      <div className="mt-10 max-md:mt-8">
        <LiveDemo />
      </div>
    </section>
  );
}
