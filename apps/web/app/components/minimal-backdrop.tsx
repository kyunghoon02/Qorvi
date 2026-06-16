/**
 * Lightweight CSS-only backdrop for inner pages.
 * Soft gradient + masked grid + two static glow orbs.
 * The animated NetworkBackground is reserved for the landing page only.
 */
export function MinimalBackdrop() {
  return (
    <div className="page-shell-backdrop" aria-hidden="true">
      <div className="page-shell-backdrop-grid" />
      <div className="page-shell-backdrop-glow page-shell-backdrop-glow--a" />
      <div className="page-shell-backdrop-glow page-shell-backdrop-glow--b" />
    </div>
  );
}
