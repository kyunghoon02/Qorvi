import type { Metadata } from "next";
import type { ReactNode } from "react";

import "@xyflow/react/dist/style.css";
import "./globals.css";

import { ClerkAuthChrome } from "./components/clerk-auth-chrome";

export const metadata: Metadata = {
  title: "Qorvi",
  description: "Qorvi workspace for wallet intelligence exploration.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: ReactNode;
}>) {
  return (
    <html lang="en">
      <body>
        <ClerkAuthChrome>{children}</ClerkAuthChrome>
      </body>
    </html>
  );
}
