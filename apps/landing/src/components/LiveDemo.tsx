import { useEffect, useRef, useState } from "react";
import { hl } from "../code";

const GO_TYPES = ["string", "int", "bool", "[]string", "time.Time", "float64"];
const TS_OF: Record<string, string> = {
  string: "string",
  int: "number",
  bool: "boolean",
  "[]string": "string[]",
  "time.Time": "string",
  float64: "number",
};

function snake(name: string) {
  return name
    .replace(/([a-z0-9])([A-Z])/g, "$1_$2")
    .replace(/([A-Z]+)([A-Z][a-z])/g, "$1_$2")
    .toLowerCase();
}

function clean(raw: string) {
  const s = raw.replace(/[^A-Za-z0-9_]/g, "").slice(0, 20);
  return s ? s[0].toUpperCase() + s.slice(1) : "";
}

const wait = (ms: number) => new Promise((r) => setTimeout(r, ms));

export function LiveDemo() {
  const [name, setName] = useState("Hello");
  const [typeIdx, setTypeIdx] = useState(0);
  const [required, setRequired] = useState(true);
  const [typing, setTyping] = useState(false);
  const [hint, setHint] = useState(false);
  const [flash, setFlash] = useState(0);
  const cancel = useRef(false);

  const rename = (n: string) => {
    setName(n);
    setFlash((f) => f + 1);
  };

  useEffect(() => {
    cancel.current = false;
    const cancelled = () => cancel.current;
    const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

    (async () => {
      if (reduce) {
        rename("Message");
        setHint(true);
        return;
      }
      await wait(1400);
      if (cancelled()) return;
      setTyping(true);
      for (let i = 4; i >= 0; i--) {
        await wait(75);
        if (cancelled()) return;
        rename("Hello".slice(0, i));
      }
      await wait(260);
      const target = "Message";
      for (let i = 1; i <= target.length; i++) {
        await wait(90);
        if (cancelled()) return;
        rename(target.slice(0, i));
      }
      await wait(600);
      if (cancelled()) return;
      setTyping(false);
      setHint(true);
    })();

    return () => {
      cancel.current = true;
    };
  }, []);

  const goType = GO_TYPES[typeIdx];
  const tsType = TS_OF[goType];
  const key = snake(name);
  const usesTime = goType === "time.Time";

  return (
    <div>
      <div className="demo" aria-live="polite">
        <div className="pane go">
          <div className="pane-head">
            <span>main.go</span>
            <span className="note">you write this</span>
          </div>
          <pre>
            <code>
              {usesTime && <>{hl('import "time"\n\n', "go")}</>}
              {hl("type ", "go")}
              <span className="tok-ty">Greeting</span>
              {hl(" struct {\n\t", "go")}
              <input
                className="demo-edit"
                aria-label="Go field name"
                value={name}
                spellCheck={false}
                autoComplete="off"
                style={{ width: `${Math.max(name.length, 1)}ch` }}
                onFocus={() => {
                  cancel.current = true;
                  setTyping(false);
                  setHint(true);
                }}
                onChange={(e) => rename(clean(e.target.value))}
              />
              {typing && <span className="caret" aria-hidden="true" />}{" "}
              <button
                type="button"
                className="demo-btn"
                aria-label={`Field type is ${goType}. Change type.`}
                onClick={() => {
                  setTypeIdx((i) => (i + 1) % GO_TYPES.length);
                  setFlash((f) => f + 1);
                }}
              >
                {goType}
              </button>
              {" `"}
              <span className="tok-str">json:"{key || "?"}"</span>{" "}
              <button
                type="button"
                className={`demo-btn ${required ? "tok-str" : "off"}`}
                aria-pressed={required}
                aria-label="Toggle the required validation tag"
                onClick={() => {
                  setRequired((r) => !r);
                  setFlash((f) => f + 1);
                }}
              >
                {required ? 'validate:"required"' : "no validate tag"}
              </button>
              {"`\n"}
              {hl("\tSentAt time.Time `json:\"sent_at\"`\n}\n\n", "go")}
              {hl("func greet(\n\tr *http.Request, in *Person,\n) (*", "go", ["Person"])}
              <span className="tok-ty">Greeting</span>
              {hl(", error) {\n\t// ...\n}\n\napi.Handle(\"POST /greet\", greet)", "go")}
            </code>
          </pre>
        </div>
        <div className="seam" aria-hidden="true">
          »
        </div>
        <div className="pane ts">
          <div className="pane-head">
            <span>api.d.ts</span>
            <span className="note">regenerated on every save</span>
          </div>
          <pre>
            <code>
              {hl("export interface ", "ts")}
              <span className="tok-ty">Greeting</span>
              {" {\n  "}
              <span key={flash} className={flash ? "line-flash" : undefined}>
                {key || <span className="tok-cm">{"/* field has no name */"}</span>}
                {key && !required && "?"}
                {key && ": "}
                {key && <span className="tok-ty">{tsType}</span>}
                {key && ";"}
              </span>
              {"\n  sent_at?: "}
              <span className="tok-ty">string</span>
              {";\n}\n\n"}
              {hl('const { data } = await client.POST(\n  "/greet", { body },\n);\n', "ts")}
              {"data."}
              {key || "…"}
              {"\n"}
              <span className="tok-cm">
                {"//   ^? "}{key ? (required ? tsType : `${tsType} | undefined`) : "unknown"}
              </span>
            </code>
          </pre>
        </div>
      </div>
      <p className="hint mt-4" style={{ opacity: hint ? 1 : 0 }} aria-hidden={!hint}>
        Your turn. Rename the field, click the type to change it, or drop the validation tag.
      </p>
    </div>
  );
}
