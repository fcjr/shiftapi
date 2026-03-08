import { useEffect, useState } from "react";
import { createHighlighterCore, type ThemedToken } from "@shikijs/core";
import { createJavaScriptRegexEngine } from "@shikijs/engine-javascript";
import langGo from "@shikijs/langs/go";
import langTypescript from "@shikijs/langs/typescript";
import themeTokyoNight from "@shikijs/themes/tokyo-night";
import { Reveal } from "./Reveal";

const goCode = `type Input struct {
    Name string \`json:"name"\`
}

type Output struct {
    Hello string \`json:"hello"\`
}

func greet(r *http.Request, in Input) (*Output, error) {
    return &Output{Hello: in.Name}, nil
}

func main() {
    api := shiftapi.New()
    shiftapi.Post(api, "/greet", greet)
    shiftapi.ListenAndServe(":8080", api)
}`;

const tsCode = `import { client } from "@shiftapi/client";

// Fully typed — inferred from your Go structs
const { data } = await client.POST("/greet", {
    body: { name: "frank" },
});

console.log(data.hello);
//          ^? (property) hello: string`;

const highlighterPromise = createHighlighterCore({
  themes: [themeTokyoNight],
  langs: [langGo, langTypescript],
  engine: createJavaScriptRegexEngine(),
});

function useTokens(code: string, lang: "go" | "typescript") {
  const [tokens, setTokens] = useState<ThemedToken[][] | null>(null);

  useEffect(() => {
    highlighterPromise.then((highlighter) => {
      const result = highlighter.codeToTokens(code, {
        lang,
        theme: "tokyo-night",
      });
      setTokens(result.tokens);
    });
  }, [code, lang]);

  return tokens;
}

function CodeBlock({ filename, dotColor, code, lang }: {
  filename: string;
  dotColor: string;
  code: string;
  lang: "go" | "typescript";
}) {
  const tokens = useTokens(code, lang);

  return (
    <div className="bg-surface border border-border rounded-2xl overflow-hidden transition-[border-color] duration-300 hover:border-border-hover min-w-0">
      <div className="flex items-center gap-2 px-[18px] py-3 border-b border-border bg-elevated">
        <span className="w-2.5 h-2.5 rounded-full shrink-0" style={{ background: dotColor }} />
        <span className="text-[13px] text-text-secondary font-mono">{filename}</span>
      </div>
      <div className="min-h-[80px]">
        {tokens && (
          <pre className="p-[22px] m-0 overflow-x-auto text-[13px] leading-[1.75]">
            <code>
              {tokens.map((line, i) => (
                <span key={i}>
                  {line.map((token, j) => (
                    <span key={j} style={{ color: token.color }}>
                      {token.content}
                    </span>
                  ))}
                  {i < tokens.length - 1 && "\n"}
                </span>
              ))}
            </code>
          </pre>
        )}
      </div>
    </div>
  );
}

export function CodeSection() {
  return (
    <section className="px-6 pb-[120px]">
      <Reveal className="text-center mb-12">
        <h2 className="text-[clamp(28px,4vw,40px)] font-extrabold tracking-[-0.03em] mb-3">
          Write Go. Get TypeScript. Done.
        </h2>
        <p className="text-[17px] text-text-secondary max-w-[520px] mx-auto">
          Your Go struct becomes the TypeScript type. Change a field in Go, your frontend knows instantly.
        </p>
      </Reveal>
      <Reveal className="max-w-[1060px] mx-auto grid grid-cols-[1fr_auto_1fr] items-center max-md:grid-cols-1">
        <CodeBlock filename="main.go" dotColor="#00ADD8" code={goCode} lang="go" />
        <div className="flex flex-col items-center gap-1.5 px-4 max-md:flex-row max-md:py-3 max-md:px-0">
          <div className="w-px h-10 bg-gradient-to-b from-transparent via-accent-bright to-transparent max-md:w-10 max-md:h-px max-md:bg-gradient-to-r" />
          <div className="w-1.5 h-1.5 rounded-full bg-accent-bright shadow-[0_0_10px_var(--color-accent-bright)]" />
          <div className="text-[10px] uppercase tracking-[0.1em] text-accent-bright [writing-mode:vertical-rl] rotate-180 whitespace-nowrap max-md:[writing-mode:horizontal-tb] max-md:rotate-0">
            auto-generated
          </div>
          <div className="w-1.5 h-1.5 rounded-full bg-accent-bright shadow-[0_0_10px_var(--color-accent-bright)]" />
          <div className="w-px h-10 bg-gradient-to-b from-transparent via-accent-bright to-transparent max-md:w-10 max-md:h-px max-md:bg-gradient-to-r" />
        </div>
        <CodeBlock filename="app.ts" dotColor="#3178C6" code={tsCode} lang="typescript" />
      </Reveal>
    </section>
  );
}
