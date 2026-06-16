import type { Metadata } from "next";
import { Inter, JetBrains_Mono } from "next/font/google";
import type { ReactNode } from "react";

import "@xyflow/react/dist/style.css";
import "./globals.css";

import { ClerkAuthChrome } from "./components/clerk-auth-chrome";

// DESIGN.md (Spotify-inspired) is sans-only. Inter is the closest free
// substitute for SpotifyMixUI / CircularSp; we load 400 + 700 (Spotify's
// bold/regular binary) to keep first-paint payload small. JetBrains Mono
// powers code chips. The serif font that the previous DESIGN.md required
// has been removed — saved ~6 font files on first paint.
const inter = Inter({
  subsets: ["latin"],
  weight: ["400", "500", "600", "700"],
  variable: "--font-sans",
  display: "swap",
});

const jetbrainsMono = JetBrains_Mono({
  subsets: ["latin"],
  weight: ["400", "500"],
  variable: "--font-mono",
  display: "swap",
});

export const metadata: Metadata = {
  title: "Qorvi",
  description: "Qorvi product scaffold for wallet intelligence exploration.",
};

import { getLocaleCookie } from "../lib/i18n/actions";
import { getDictionary } from "../lib/i18n/dictionaries";
import { I18nProvider } from "../lib/i18n/provider";

export default async function RootLayout({
  children,
}: Readonly<{
  children: ReactNode;
}>) {
  const locale = await getLocaleCookie();
  const dictionary = getDictionary(locale);

  return (
    <html
      lang={locale}
      className={`${inter.variable} ${jetbrainsMono.variable}`}
    >
      <body>
        <ClerkAuthChrome>
          <I18nProvider locale={locale} dictionary={dictionary}>
            {children}
          </I18nProvider>
        </ClerkAuthChrome>
      </body>
    </html>
  );
}
