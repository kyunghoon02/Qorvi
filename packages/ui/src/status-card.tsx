import type { CSSProperties, ReactNode } from "react";

export type Tone = "teal" | "amber" | "violet" | "emerald";

type ToneTokens = {
  badgeBg: string;
  badgeText: string;
  border: string;
  glow: string;
  heading: string;
  accent: string;
};

const toneTokens: Record<Tone, ToneTokens> = {
  teal: {
    badgeBg: "rgba(77, 227, 194, 0.14)",
    badgeText: "#92f0dd",
    border: "rgba(77, 227, 194, 0.22)",
    glow: "rgba(77, 227, 194, 0.1)",
    heading: "#f2fffb",
    accent: "#4de3c2",
  },
  amber: {
    badgeBg: "rgba(246, 196, 108, 0.16)",
    badgeText: "#ffd98b",
    border: "rgba(246, 196, 108, 0.24)",
    glow: "rgba(246, 196, 108, 0.12)",
    heading: "#fff7e6",
    accent: "#f6c46c",
  },
  violet: {
    badgeBg: "rgba(183, 166, 255, 0.16)",
    badgeText: "#d6ceff",
    border: "rgba(183, 166, 255, 0.24)",
    glow: "rgba(183, 166, 255, 0.12)",
    heading: "#f6f3ff",
    accent: "#b7a6ff",
  },
  emerald: {
    badgeBg: "rgba(136, 227, 141, 0.14)",
    badgeText: "#bdffbf",
    border: "rgba(136, 227, 141, 0.22)",
    glow: "rgba(136, 227, 141, 0.12)",
    heading: "#f1fff2",
    accent: "#88e38d",
  },
};

const cardBase: CSSProperties = {
  borderRadius: 24,
  border: "1px solid transparent",
  background:
    "linear-gradient(180deg, rgba(17, 25, 42, 0.96) 0%, rgba(10, 15, 27, 0.94) 100%)",
  boxShadow: "0 20px 50px rgba(0, 0, 0, 0.22)",
  padding: 20,
};

function getTone(tone: Tone): ToneTokens {
  return toneTokens[tone];
}

export function Pill({
  children,
  tone,
}: {
  children: ReactNode;
  tone: Tone;
}) {
  const palette = getTone(tone);

  return (
    <span
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: 8,
        padding: "8px 12px",
        borderRadius: 999,
        border: `1px solid ${palette.border}`,
        background: palette.badgeBg,
        color: palette.badgeText,
        fontSize: 12,
        fontWeight: 700,
        letterSpacing: "0.08em",
        textTransform: "uppercase",
      }}
    >
      {children}
    </span>
  );
}

export function Badge({
  children,
  tone,
}: {
  children: ReactNode;
  tone: Tone;
}) {
  return <Pill tone={tone}>{children}</Pill>;
}

export function MetricCard({
  label,
  value,
  hint,
  tone,
}: {
  label: string;
  value: string;
  hint: string;
  tone: Tone;
}) {
  const palette = getTone(tone);

  return (
    <article
      style={{
        ...cardBase,
        background: `linear-gradient(180deg, ${palette.glow} 0%, rgba(10, 15, 27, 0.94) 42%)`,
        borderColor: palette.border,
        minHeight: 132,
      }}
    >
      <div
        style={{
          color: palette.badgeText,
          fontSize: 12,
          textTransform: "uppercase",
          letterSpacing: "0.18em",
        }}
      >
        {label}
      </div>
      <div
        style={{
          marginTop: 12,
          fontSize: 30,
          fontWeight: 700,
          letterSpacing: "-0.04em",
          color: palette.heading,
        }}
      >
        {value}
      </div>
      <p
        style={{
          margin: "10px 0 0",
          color: "rgba(233, 240, 255, 0.68)",
          lineHeight: 1.45,
        }}
      >
        {hint}
      </p>
    </article>
  );
}

export function StatusCard({
  eyebrow,
  title,
  summary,
  badgeLabel,
  bullets,
  tags,
  footer,
  tone,
}: {
  eyebrow: string;
  title: string;
  summary: string;
  badgeLabel: string;
  bullets: readonly string[];
  tags: readonly string[];
  footer: string;
  tone: Tone;
}) {
  const palette = getTone(tone);

  return (
    <article
      style={{
        ...cardBase,
        borderColor: palette.border,
        background: `linear-gradient(180deg, ${palette.glow} 0%, rgba(10, 15, 27, 0.94) 48%)`,
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          gap: 12,
          marginBottom: 14,
        }}
      >
        <span
          style={{
            color: "rgba(233, 240, 255, 0.66)",
            fontSize: 12,
            textTransform: "uppercase",
            letterSpacing: "0.2em",
          }}
        >
          {eyebrow}
        </span>
        <Badge tone={tone}>{badgeLabel}</Badge>
      </div>

      <h3
        style={{
          margin: 0,
          color: palette.heading,
          fontSize: 22,
          lineHeight: 1.1,
          letterSpacing: "-0.03em",
        }}
      >
        {title}
      </h3>
      <p
        style={{
          margin: "12px 0 0",
          color: "rgba(233, 240, 255, 0.7)",
          lineHeight: 1.6,
        }}
      >
        {summary}
      </p>

      <ul
        style={{
          margin: "16px 0 0",
          padding: 0,
          listStyle: "none",
          display: "grid",
          gap: 10,
        }}
      >
        {bullets.map((bullet) => (
          <li
            key={bullet}
            style={{
              display: "grid",
              gridTemplateColumns: "auto 1fr",
              gap: 10,
              alignItems: "start",
              color: "rgba(233, 240, 255, 0.84)",
              lineHeight: 1.55,
            }}
          >
            <span
              aria-hidden="true"
              style={{
                marginTop: 9,
                width: 7,
                height: 7,
                borderRadius: 999,
                background: palette.accent,
              }}
            />
            <span>{bullet}</span>
          </li>
        ))}
      </ul>

      <div
        style={{
          display: "flex",
          flexWrap: "wrap",
          gap: 8,
          marginTop: 16,
        }}
      >
        {tags.map((tag) => (
          <span
            key={tag}
            style={{
              padding: "6px 10px",
              borderRadius: 999,
              border: "1px solid rgba(148, 163, 184, 0.16)",
              background: "rgba(255, 255, 255, 0.02)",
              color: "rgba(233, 240, 255, 0.72)",
              fontSize: 12,
              textTransform: "uppercase",
              letterSpacing: "0.08em",
            }}
          >
            {tag}
          </span>
        ))}
      </div>

      <p
        style={{
          margin: "16px 0 0",
          paddingTop: 14,
          borderTop: "1px solid rgba(148, 163, 184, 0.12)",
          color: "rgba(233, 240, 255, 0.6)",
          fontSize: 13,
        }}
      >
        {footer}
      </p>
    </article>
  );
}
