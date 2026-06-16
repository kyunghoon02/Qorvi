"use client";

import Link from "next/link";

import { useTranslation } from "../lib/i18n/provider";
import { PageShell } from "./components/page-shell";

export function LandingScreen() {
  const { dictionary } = useTranslation();
  const l = dictionary.landing;

  return (
    <PageShell background="network">
      <section className="landing-hero">
        <div className="landing-hero-copy">
          <span className="landing-hero-eyebrow">{l.eyebrow}</span>
          <h1>
            {l.headlineLeft}
            <br />
            <em>{l.headlineRight}</em>
          </h1>
          <p>{l.sub}</p>
          <div className="landing-hero-cta">
            <Link className="landing-cta-primary" href="/copilot">
              {l.ctaPrimary}
            </Link>
            <Link className="landing-cta-secondary" href="/discover">
              {l.ctaSecondary}
            </Link>
          </div>
        </div>
      </section>
    </PageShell>
  );
}
