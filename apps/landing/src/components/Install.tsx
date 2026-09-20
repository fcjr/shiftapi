import { useState } from "react";
import { CheckIcon, CopyIcon } from "../icons";

const CMD = "npm create shiftapi@latest";

export function Install() {
  const [copied, setCopied] = useState(false);

  const copy = () => {
    navigator.clipboard.writeText(CMD);
    setCopied(true);
    setTimeout(() => setCopied(false), 1800);
  };

  return (
    <div className="cmd">
      <span className="prompt">$</span>
      <code>{CMD}</code>
      <button type="button" aria-label={copied ? "Copied" : "Copy command"} onClick={copy}>
        {copied ? <CheckIcon className="text-go" /> : <CopyIcon />}
      </button>
    </div>
  );
}
