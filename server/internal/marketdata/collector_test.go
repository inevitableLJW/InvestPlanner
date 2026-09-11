package marketdata

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"investplanner/server/internal/database"
)

type collectorTestStore struct {
	shares []database.FundShare
}

func (s *collectorTestStore) ListActiveFundShares(at time.Time) ([]database.FundShare, error) {
	result := []database.FundShare{}
	for _, share := range s.shares {
		if share.Active && !share.EffectiveFrom.After(at) {
			result = append(result, share)
		}
	}
	return result, nil
}
func (s *collectorTestStore) ListActiveBenchmarkMappings(time.Time) ([]database.BenchmarkMapping, error) {
	return nil, nil
}
func (s *collectorTestStore) UpsertVerifiedFundShareMapping(share database.FundShare, _ string, effectiveAt time.Time) error {
	// Simulate MySQL rounding the persisted timestamp just past the collector's
	// in-memory timestamp. A re-query in the same refresh would exclude it.
	share.EffectiveFrom = effectiveAt.Add(time.Millisecond)
	s.shares = append(s.shares, share)
	return nil
}
func (s *collectorTestStore) UpsertDiscoveredFundShare(database.FundShare) error { return nil }

type collectorSnapshotStore struct {
	snapshots []database.MarketDataSnapshot
}

func (s *collectorSnapshotStore) SaveSnapshot(snapshot database.MarketDataSnapshot) error {
	s.snapshots = append(s.snapshots, snapshot)
	return nil
}
func (s *collectorSnapshotStore) GetSnapshot(string, string) (database.MarketDataSnapshot, error) {
	return database.MarketDataSnapshot{}, errors.New("not found")
}

func TestRefreshPurchasesSnapshotsNewMappingsInFirstRun(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimPrefix(r.URL.Path, "/api/public/") {
		case "fund_open_fund_rank_em":
			_, _ = w.Write([]byte(`[{"基金代码":"000834","基金简称":"大成纳斯达克100ETF联接(QDII)A"}]`))
		case "fund_purchase_em":
			_, _ = w.Write([]byte(`[{"基金代码":"000834","基金简称":"大成纳斯达克100ETF联接(QDII)A","申购状态":"限大额","购买起点":10,"日累计限定金额":10}]`))
		case "fund_overview_em":
			_, _ = w.Write([]byte(`[{"基金代码":"000834","跟踪标的":"纳斯达克100指数"}]`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	now := time.Date(2026, 8, 26, 1, 22, 0, 741500000, time.UTC)
	store := &collectorTestStore{}
	snapshotStore := &collectorSnapshotStore{}
	client := NewAKToolsClient(upstream.URL, time.Second)
	client.Now = func() time.Time { return now }
	collector := &Collector{
		Store:     store,
		Snapshots: &SnapshotService{Store: snapshotStore, Now: func() time.Time { return now }},
		AKTools:   client,
		Now:       func() time.Time { return now },
	}
	if err := collector.refreshPurchases(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(snapshotStore.snapshots) != 1 || snapshotStore.snapshots[0].DataType != "purchase_limit" || snapshotStore.snapshots[0].EntityKey != "000834" {
		t.Fatalf("snapshots=%+v", snapshotStore.snapshots)
	}
}

func TestRefreshPurchasesWritesExistingFundsBeforeDiscoveringNewMappings(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimPrefix(r.URL.Path, "/api/public/") {
		case "fund_open_fund_rank_em":
			// The daily ranking catalog can omit an already verified fund. It is
			// still authoritative for discovering the brand-new code only.
			_, _ = w.Write([]byte(`[{
				"基金代码":"000834","基金简称":"大成纳斯达克100ETF联接(QDII)A"
			}]`))
		case "fund_purchase_em":
			// Put the new mapping first to prove input order cannot delay an
			// already verified fund's snapshot.
			_, _ = w.Write([]byte(`[
				{"基金代码":"000834","基金简称":"大成纳斯达克100ETF联接(QDII)A","申购状态":"限大额","购买起点":10,"日累计限定金额":10},
				{"基金代码":"270042","基金简称":"广发纳斯达克100ETF联接人民币(QDII)A","申购状态":"限大额","购买起点":10,"日累计限定金额":5}
			]`))
		case "fund_overview_em":
			_, _ = w.Write([]byte(`[{"基金代码":"000834","跟踪标的":"纳斯达克100指数"}]`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	now := time.Date(2026, 8, 26, 1, 22, 0, 0, time.UTC)
	store := &collectorTestStore{shares: []database.FundShare{{FundCode: "270042", Venue: "otc", Active: true, EffectiveFrom: now.Add(-time.Hour)}}}
	snapshotStore := &collectorSnapshotStore{}
	collector := &Collector{
		Store:     store,
		Snapshots: &SnapshotService{Store: snapshotStore, Now: func() time.Time { return now }},
		AKTools:   NewAKToolsClient(upstream.URL, time.Second),
		Now:       func() time.Time { return now },
	}
	if err := collector.refreshPurchases(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(snapshotStore.snapshots) != 2 || snapshotStore.snapshots[0].EntityKey != "270042" || snapshotStore.snapshots[1].EntityKey != "000834" {
		t.Fatalf("snapshot order=%+v", snapshotStore.snapshots)
	}
}

func TestRefreshFXFallsBackToSAFE(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"日期":"2026-08-25T00:00:00.000","美元":678.52,"港元":86.587}]`))
	}))
	defer upstream.Close()
	now := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)
	snapshotStore := &collectorSnapshotStore{}
	client := NewAKToolsClient(upstream.URL, time.Second)
	client.Now = func() time.Time { return now }
	collector := &Collector{Snapshots: &SnapshotService{Store: snapshotStore}, AKTools: client}
	if err := collector.refreshFX(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(snapshotStore.snapshots) != 2 || snapshotStore.snapshots[0].EntityKey != "USD" || snapshotStore.snapshots[1].EntityKey != "HKD" {
		t.Fatalf("snapshots=%+v", snapshotStore.snapshots)
	}
	for index, expected := range []string{`"cnyPerUnit":6.7852`, `"cnyPerUnit":0.86587`} {
		if !strings.Contains(string(snapshotStore.snapshots[index].NormalizedJSON), expected) {
			t.Fatalf("snapshot %d=%s, want %s", index, snapshotStore.snapshots[index].NormalizedJSON, expected)
		}
		if snapshotStore.snapshots[index].Provider != "safe" || snapshotStore.snapshots[index].SourceURL != SAFEReferenceRateSource {
			t.Fatalf("snapshot %d metadata=%+v", index, snapshotStore.snapshots[index])
		}
	}
}

func TestETFTrackingPrefersPublicCumulativeNAVPerFund(t *testing.T) {
	publicNAVCalls := map[string]int{}
	etfNAVCalls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("symbol")
		switch strings.TrimPrefix(r.URL.Path, "/api/public/") {
		case "fund_etf_fund_info_em":
			etfNAVCalls++
			_, _ = w.Write([]byte(`[]`))
		case "fund_open_fund_info_em":
			if r.URL.Query().Get("indicator") != "累计净值走势" {
				t.Fatalf("indicator=%q", r.URL.Query().Get("indicator"))
			}
			publicNAVCalls[code]++
			if code == "513100" && publicNAVCalls[code] == 1 {
				http.Error(w, "temporary upstream failure", http.StatusBadGateway)
				return
			}
			_, _ = w.Write([]byte(`[{"净值日期":"2026-09-03T00:00:00.000","累计净值":1.20},{"净值日期":"2026-09-04T00:00:00.000","累计净值":1.21}]`))
		default:
			t.Fatalf("must not fall back to price after public NAV succeeds: %s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	client := NewAKToolsClient(upstream.URL, time.Second)
	client.Now = func() time.Time { return time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC) }
	collector := &Collector{AKTools: client}
	for _, code := range []string{"513100", "513500"} {
		series, kind, err := collector.freeTrackingSeries(context.Background(), database.FundShare{FundCode: code, Venue: "etf"})
		if err != nil || kind != "fund_etf_tracking_series" || series.Basis != "cumulative_nav_proxy" || len(series.Points) != 2 {
			t.Fatalf("code=%s series=%+v kind=%s err=%v", code, series, kind, err)
		}
	}
	if publicNAVCalls["513100"] != 2 || publicNAVCalls["513500"] != 1 {
		t.Fatalf("public NAV calls=%v", publicNAVCalls)
	}
	if etfNAVCalls != 0 {
		t.Fatalf("ETF-specific NAV fallback calls=%d, want 0", etfNAVCalls)
	}
}

func TestETFTrackingDoesNotSubstituteMarketPriceForMissingNAV(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimPrefix(r.URL.Path, "/api/public/") {
		case "fund_etf_fund_info_em", "fund_open_fund_info_em":
			_, _ = w.Write([]byte(`[]`))
		default:
			t.Fatalf("market-price route must not be used as tracking error: %s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	collector := &Collector{AKTools: NewAKToolsClient(upstream.URL, time.Second)}
	series, _, err := collector.freeTrackingSeries(context.Background(), database.FundShare{FundCode: "513100", Venue: "etf"})
	if err == nil || series.Basis != "" {
		t.Fatalf("series=%+v err=%v", series, err)
	}
}
