import type { Metadata } from "next";
import type { ReactNode } from "react";

import "@xyflow/react/dist/style.css";
import "./globals.css";

import { ClerkAuthChrome } from "./components/clerk-auth-chrome";

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
    <html lang={locale}>
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
