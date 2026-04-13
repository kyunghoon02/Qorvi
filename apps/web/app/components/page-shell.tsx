"use client";

import type { ReactNode } from "react";

import { AuthButtons } from "./auth-buttons";
import { LanguageSwitcher } from "./language-switcher";
import { NetworkBackground } from "./network-background";

type NavItem = {
  href: string;
  label: string;
  matchPrefix: string;
};

const navItems: NavItem[] = [
  { href: "/discover", label: "Discover", matchPrefix: "/discover" },
  { href: "/signals/shadow-exits", label: "Signals", matchPrefix: "/signals" },
  { href: "/alerts", label: "Alerts", matchPrefix: "/alerts" },
];

/**
 * Shared page shell used by all sub-pages.
 * Provides consistent header navigation, the animated NetworkBackground,
 * and a centered content area matching the home/discover layout.
 *
 * @param activeRoute  — current route prefix so the nav highlights correctly
 * @param children     — page-specific content
 */
export function PageShell({
  activeRoute,
  children,
}: {
  activeRoute?: string;
  children: ReactNode;
}) {
  return (
    <div className="page-shell-layout">
      <NetworkBackground />

      <header className="home-fullscreen-header">
        <div className="home-fullscreen-brand">
          <h1
            style={{
              fontSize: "1.1rem",
              fontWeight: 600,
              letterSpacing: "-0.01em",
              margin: 0,
            }}
          >
            <a href="/" style={{ textDecoration: "none", color: "inherit" }}>
              Qorvi
            </a>
          </h1>
          <nav className="discover-nav">
            {navItems.map((item) => (
              <a
                key={item.href}
                href={item.href}
                className={`discover-nav-link${
                  activeRoute?.startsWith(item.matchPrefix)
                    ? " discover-nav-link--active"
                    : ""
                }`}
              >
                {item.label}
              </a>
            ))}
          </nav>
        </div>
        <div
          style={{
            marginLeft: "auto",
            display: "flex",
            alignItems: "center",
            gap: "12px",
          }}
        >
          <LanguageSwitcher />
          <AuthButtons />
        </div>
      </header>

      <main className="page-shell-content">{children}</main>
    </div>
  );
}
