import { useEffect, useRef, useState } from "react";

export function useStarCount() {
  const [count, setCount] = useState<number | null>(null);

  useEffect(() => {
    fetch("https://api.github.com/repos/fcjr/shiftapi")
      .then((r) => r.json())
      .then((data) => {
        if (typeof data.stargazers_count === "number") {
          setCount(data.stargazers_count);
        }
      })
      .catch(() => {});
  }, []);

  return count;
}

export function useCountUp(target: number | null, duration = 600) {
  const [display, setDisplay] = useState<string>("");

  useEffect(() => {
    if (target === null) return;
    if (target === 0) {
      setDisplay("0");
      return;
    }
    const start = performance.now();
    let raf: number;
    const step = (now: number) => {
      const progress = Math.min((now - start) / duration, 1);
      const eased = 1 - Math.pow(1 - progress, 3);
      setDisplay(Math.round(eased * target).toLocaleString());
      if (progress < 1) raf = requestAnimationFrame(step);
    };
    raf = requestAnimationFrame(step);
    return () => cancelAnimationFrame(raf);
  }, [target, duration]);

  return display;
}

export function useReveal() {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          el.classList.add("visible");
          observer.unobserve(el);
        }
      },
      { threshold: 0.1, rootMargin: "0px 0px -40px 0px" }
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  return ref;
}
