import { PageShell } from "./page-shell";

export function ComingSoonScreen({
  title,
}: {
  title: string;
}) {
  return (
    <PageShell background="none">
      <section className="copilot-empty">
        <p className="detail-limitations-kicker">Coming Soon</p>
        <h1>{title}</h1>
        <p>
          This surface is outside the current Ethereum Wallet Copilot beta
          scope. Wallet search, flow evidence, and Copilot analysis are
          available now.
        </p>
        <a className="copilot-primary" href="/copilot">
          Open Copilot
        </a>
      </section>
    </PageShell>
  );
}
