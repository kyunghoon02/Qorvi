"use client";

import { type ReactNode, createContext, useContext } from "react";
import type { Dictionary, Locale } from "./dictionaries";

export type I18nContextType = {
  locale: Locale;
  dictionary: Dictionary;
};

const I18nContext = createContext<I18nContextType | null>(null);

export function I18nProvider({
  locale,
  dictionary,
  children,
}: {
  locale: Locale;
  dictionary: Dictionary;
  children: ReactNode;
}) {
  return (
    <I18nContext.Provider value={{ locale, dictionary }}>
      {children}
    </I18nContext.Provider>
  );
}

export function useTranslation() {
  const context = useContext(I18nContext);
  if (!context) {
    throw new Error("useTranslation must be used within an I18nProvider");
  }

  const { dictionary, locale } = context;

  // Simple nested path resolver (e.g. 'hero.title')
  const t = (path: string) => {
    const keys = path.split(".");
    let current: unknown = dictionary;
    for (const key of keys) {
      if (
        typeof current !== "object" ||
        current === null ||
        !(key in current)
      ) {
        return path;
      }
      current = (current as Record<string, unknown>)[key];
    }
    return typeof current === "string" ? current : path;
  };

  return { t, locale, dictionary };
}
