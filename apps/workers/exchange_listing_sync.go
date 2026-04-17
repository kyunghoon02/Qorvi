package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/providers"
)

const workerModeExchangeListingRegistrySync = "exchange-listing-registry-sync"

type ExchangeListingRegistrySyncService struct {
	Store   db.ExchangeListingRegistryStore
	JobRuns db.JobRunStore
	Upbit   interface {
		FetchUpbitListings(context.Context) ([]providers.ExchangeListing, error)
	}
	Bithumb interface {
		FetchBithumbListings(context.Context) ([]providers.ExchangeListing, error)
	}
	Now func() time.Time
}

type ExchangeListingRegistrySyncReport struct {
	UpbitListings   int
	BithumbListings int
	ExchangesSynced []string
}

func (s ExchangeListingRegistrySyncService) RunSync(
	ctx context.Context,
) (ExchangeListingRegistrySyncReport, error) {
	if s.Store == nil {
		return ExchangeListingRegistrySyncReport{}, fmt.Errorf("exchange listing registry store is required")
	}
	if s.Upbit == nil {
		return ExchangeListingRegistrySyncReport{}, fmt.Errorf("upbit listing client is required")
	}
	if s.Bithumb == nil {
		return ExchangeListingRegistrySyncReport{}, fmt.Errorf("bithumb listing client is required")
	}

	startedAt := s.now().UTC()

	upbitListings, err := s.Upbit.FetchUpbitListings(ctx)
	if err != nil {
		s.recordJobRun(ctx, startedAt, db.JobRunStatusFailed, map[string]any{"error": err.Error(), "exchange": "upbit"})
		return ExchangeListingRegistrySyncReport{}, err
	}
	bithumbListings, err := s.Bithumb.FetchBithumbListings(ctx)
	if err != nil {
		s.recordJobRun(ctx, startedAt, db.JobRunStatusFailed, map[string]any{"error": err.Error(), "exchange": "bithumb"})
		return ExchangeListingRegistrySyncReport{}, err
	}

	observedAt := s.now().UTC()
	entries := append(
		buildExchangeListingRegistryEntries(upbitListings, observedAt),
		buildExchangeListingRegistryEntries(bithumbListings, observedAt)...,
	)
	if err := s.Store.UpsertExchangeListings(ctx, entries); err != nil {
		s.recordJobRun(ctx, startedAt, db.JobRunStatusFailed, map[string]any{"error": err.Error()})
		return ExchangeListingRegistrySyncReport{}, err
	}

	report := ExchangeListingRegistrySyncReport{
		UpbitListings:   len(upbitListings),
		BithumbListings: len(bithumbListings),
		ExchangesSynced: compactSyncedExchanges(len(upbitListings), len(bithumbListings)),
	}
	s.recordJobRun(ctx, startedAt, db.JobRunStatusSucceeded, map[string]any{
		"upbit_listings":   report.UpbitListings,
		"bithumb_listings": report.BithumbListings,
		"exchanges":        report.ExchangesSynced,
	})
	return report, nil
}

func buildExchangeListingRegistrySyncSummary(report ExchangeListingRegistrySyncReport) string {
	exchanges := "none"
	if len(report.ExchangesSynced) > 0 {
		exchanges = strings.Join(report.ExchangesSynced, ",")
	}
	return fmt.Sprintf(
		"Exchange listing registry sync complete (exchanges=%s, upbit=%d, bithumb=%d)",
		exchanges,
		report.UpbitListings,
		report.BithumbListings,
	)
}

func compactSyncedExchanges(upbitCount, bithumbCount int) []string {
	exchanges := make([]string, 0, 2)
	if upbitCount > 0 {
		exchanges = append(exchanges, "upbit")
	}
	if bithumbCount > 0 {
		exchanges = append(exchanges, "bithumb")
	}
	sort.Strings(exchanges)
	return exchanges
}

func buildExchangeListingRegistryEntries(
	listings []providers.ExchangeListing,
	observedAt time.Time,
) []db.ExchangeListingRegistryEntry {
	entries := make([]db.ExchangeListingRegistryEntry, 0, len(listings))
	for _, listing := range listings {
		entries = append(entries, db.ExchangeListingRegistryEntry{
			Exchange:           string(listing.Exchange),
			Market:             listing.Market,
			BaseSymbol:         listing.BaseSymbol,
			QuoteSymbol:        listing.QuoteSymbol,
			DisplayName:        listing.DisplayName,
			MarketWarning:      listing.MarketWarning,
			NormalizedAssetKey: listing.NormalizedAssetKey,
			TokenAddress:       listing.TokenAddress,
			ChainHint:          listing.ChainHint,
			Listed:             true,
			ListedAtDetected:   observedAt,
			LastCheckedAt:      observedAt,
			Metadata:           listing.Metadata,
		})
	}
	return entries
}

func (s ExchangeListingRegistrySyncService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s ExchangeListingRegistrySyncService) recordJobRun(
	ctx context.Context,
	startedAt time.Time,
	status db.JobRunStatus,
	details map[string]any,
) {
	if s.JobRuns == nil {
		return
	}
	finishedAt := s.now().UTC()
	_ = s.JobRuns.RecordJobRun(ctx, db.JobRunEntry{
		JobName:    workerModeExchangeListingRegistrySync,
		Status:     status,
		StartedAt:  startedAt,
		FinishedAt: &finishedAt,
		Details:    details,
	})
}
