import { Reveal } from "./Reveal";

const cards = [
  {
    label: "Without ShiftAPI",
    color: "red" as const,
    items: [
      "Manually write TypeScript types that mirror Go structs",
      "Run a codegen CLI every time the API changes",
      "Debug mismatches between frontend and backend at runtime",
      "Maintain a separate OpenAPI spec file by hand",
    ],
    icon: "✗",
  },
  {
    label: "With ShiftAPI",
    color: "green" as const,
    items: [
      "Types are generated directly from your Go code",
      "Changes propagate automatically via Vite/Next.js HMR",
      "Type errors caught at compile time, not production",
      "OpenAPI spec generated at runtime \u2014 always accurate",
    ],
    icon: "✓",
  },
];

const colorStyles = {
  red: { card: "border-red/12 bg-red/3", label: "text-red", icon: "text-red" },
  green: { card: "border-green/12 bg-green/3", label: "text-green", icon: "text-green" },
};

export function Contrast() {
  return (
    <Reveal className="px-6 pb-[120px]">
      <div className="max-w-[800px] mx-auto grid grid-cols-2 gap-5 max-md:grid-cols-1">
        {cards.map((card) => {
          const styles = colorStyles[card.color];
          return (
            <div key={card.label} className={`p-8 rounded-2xl border ${styles.card}`}>
              <div className={`text-xs font-bold uppercase tracking-[0.08em] mb-5 ${styles.label}`}>{card.label}</div>
              <ul className="list-none flex flex-col gap-3.5">
                {card.items.map((item) => (
                  <li key={item} className="text-sm leading-relaxed pl-6 relative text-text-secondary">
                    <span className={`absolute left-0 ${styles.icon}`}>{card.icon}</span>
                    {item}
                  </li>
                ))}
              </ul>
            </div>
          );
        })}
      </div>
    </Reveal>
  );
}
