"use client";

import dynamic from "next/dynamic";
import type { ReactNode } from "react";

import { AuthButtons } from "./auth-buttons";
import { LanguageSwitcher } from "./language-switcher";
import { MinimalBackdrop } from "./minimal-backdrop";

// NetworkBackground pulls in three.js + react-three-fiber + drei (~600KB).
// We lazy-load it so pages that don't use background="network" don't pay
// for it on first paint. ssr:false avoids hydration cost as well.
const NetworkBackground = dynamic(
  () =>
    import("./network-background").then((mod) => ({
      default: mod.NetworkBackground,
    })),
  { ssr: false, loading: () => null },
);

type NavItem = {
  href: string;
  label: string;
  matchPrefix: string;
};

const navItems: NavItem[] = [
  { href: "/copilot", label: "Copilot", matchPrefix: "/copilot" },
  { href: "/discover", label: "Discover", matchPrefix: "/discover" },
];

/**
 * Shared page shell used by all sub-pages.
 * Provides consistent header navigation, a backdrop, and a centered content
 * area matching the home/discover layout.
 *
 * @param activeRoute — current route prefix so the nav highlights correctly
 * @param background — visual treatment for the page backdrop. Defaults to
 *   "minimal" (a soft CSS gradient). Pass "network" only for the landing
 *   experience to keep the heavy animated 3D backdrop.
 * @param children — page-specific content
 */
export function PageShell({
  activeRoute,
  background = "minimal",
  children,
}: {
  activeRoute?: string;
  background?: "network" | "minimal" | "none";
  children: ReactNode;
}) {
  return (
    <div className={`page-shell-layout page-shell-layout--${background}`}>
      {background === "network" ? <NetworkBackground /> : null}
      {background === "minimal" ? <MinimalBackdrop /> : null}

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
                  (item.href === "/" && !activeRoute) ||
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
