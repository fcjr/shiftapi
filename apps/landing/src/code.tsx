import { Fragment } from "react";

type Lang = "go" | "ts";

const KEYWORDS: Record<Lang, RegExp> = {
  go: /^(package|import|func|type|struct|return|var|const|if|else|for|range|map|chan|go|defer|nil|true|false)$/,
  ts: /^(import|export|from|const|let|await|async|interface|type|return|function|new|true|false|null|undefined)$/,
};

const BUILTIN_TYPES: Record<Lang, RegExp> = {
  go: /^(string|int|int64|bool|float64|byte|error|any)$/,
  ts: /^(string|number|boolean|unknown|void|never)$/,
};

const SCAN = /(\/\/[^\n]*)|("(?:[^"\\]|\\.)*"|`[^`]*`)|(\b\d+(?:\.\d+)?\b)|([A-Za-z_][A-Za-z0-9_]*)|(\s+|[^\sA-Za-z0-9_"`]+)/g;

export function hl(code: string, lang: Lang, types: string[] = []) {
  const out: React.ReactNode[] = [];
  let i = 0;
  for (const m of code.matchAll(SCAN)) {
    const [text, comment, str, num, ident] = m;
    let cls = "";
    if (comment) cls = "tok-cm";
    else if (str) cls = "tok-str";
    else if (num) cls = "tok-num";
    else if (ident) {
      if (KEYWORDS[lang].test(ident)) cls = "tok-kw";
      else if (BUILTIN_TYPES[lang].test(ident) || types.includes(ident)) cls = "tok-ty";
    }
    out.push(cls ? <span key={i++} className={cls}>{text}</span> : <Fragment key={i++}>{text}</Fragment>);
  }
  return out;
}

export function Code({ lang, children, types }: { lang: Lang; children: string; types?: string[] }) {
  return (
    <pre className={lang}>
      <code>{hl(children, lang, types)}</code>
    </pre>
  );
}
