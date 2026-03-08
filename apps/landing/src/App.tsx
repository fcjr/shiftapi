import { Nav, Hero, Pipeline, CodeSection, Contrast, Features, CTA, Footer } from "./components";

export function App() {
  return (
    <>
      <div className="bg-grid" />
      <div className="bg-glow bg-glow-1" />
      <div className="bg-glow bg-glow-2" />
      <Nav />
      <Hero />
      <Pipeline />
      <CodeSection />
      <Contrast />
      <Features />
      <CTA />
      <Footer />
    </>
  );
}
