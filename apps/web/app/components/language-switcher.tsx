"use client";

import { useRouter } from "next/navigation";
import { useTransition } from "react";
import { setLocaleCookie } from "../../lib/i18n/actions";
import { useTranslation } from "../../lib/i18n/provider";

export function LanguageSwitcher() {
  const { locale } = useTranslation();
  const router = useRouter();
  const [isPending, startTransition] = useTransition();

  const toggleLanguage = () => {
    const nextLocale = locale === "en" ? "ko" : "en";

    startTransition(async () => {
      await setLocaleCookie(nextLocale);
      router.refresh();
    });
  };

  return (
    <button
      type="button"
      onClick={toggleLanguage}
      disabled={isPending}
      className="app-auth-button"
      style={{
        padding: "6px 14px",
        minWidth: "50px",
        fontWeight: 600,
        fontSize: "0.85rem",
        opacity: isPending ? 0.7 : 1,
      }}
      title={locale === "en" ? "Switch to Korean" : "Switch to English"}
    >
      {locale === "en" ? "EN" : "KR"}
    </button>
  );
}
