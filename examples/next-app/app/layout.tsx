import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "ShiftAPI + Next.js Example",
  description: "Built with ShiftAPI",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
