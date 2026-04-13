package server

import (
	"net/http/httptest"
	"testing"
)

func TestTierFromHeaderAcceptsQorviPlanHeader(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/healthz", nil)
	req.Header.Set("X-Qorvi-Plan", "team")

	if got := tierFromHeader(req); got != "team" {
		t.Fatalf("expected team tier, got %q", got)
	}
}

func TestTierFromHeaderAcceptsLegacyPlanHeaders(t *testing.T) {
	t.Parallel()

	for _, headerName := range []string{"X-Flowintel-Plan", "X-Whalegraph-Plan"} {
		req := httptest.NewRequest("GET", "/healthz", nil)
		req.Header.Set(headerName, "pro")

		if got := tierFromHeader(req); got != "pro" {
			t.Fatalf("expected pro tier from %s, got %q", headerName, got)
		}
	}
}
