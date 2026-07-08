import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "GateKeeper Dashboard",
  description: "Enterprise API Gateway for LLMs",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className="antialiased font-sans bg-gray-950 text-white">
        {children}
      </body>
    </html>
  );
}
