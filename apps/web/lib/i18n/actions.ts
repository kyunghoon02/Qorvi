"use server";

import { cookies } from "next/headers";
import { type Locale, defaultLocale } from "./dictionaries";

const COOKIE_NAME = "NEXT_LOCALE";

export async function setLocaleCookie(locale: Locale) {
  cookies().set(COOKIE_NAME, locale, {
    maxAge: 60 * 60 * 24 * 365, // 1 year
    path: "/",
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
  });
}

export async function getLocaleCookie(): Promise<Locale> {
  const cookieVal = cookies().get(COOKIE_NAME)?.value;
  return cookieVal === "ko" || cookieVal === "en" ? cookieVal : defaultLocale;
}
