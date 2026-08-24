package marketdata

import (
	"context"
	"time"
)

type SourceMeta struct {
	Provider   string    `json:"provider"`
	SourceURL  string    `json:"sourceUrl"`
	ObservedAt time.Time `json:"observedAt"`
	FetchedAt  time.Time `json:"fetchedAt"`
}

type FundCatalogItem struct {
	Code, Name, ShareClass string
	Meta                   SourceMeta
}
type PurchaseLimit struct {
	FundCode, Name, ShareClass    string
	Venue, Channel, Status        string
	NextOpenDate                  *time.Time
	MinimumCents, DailyLimitCents *int64
	Meta                          SourceMeta
}
type FundOverview struct {
	FundCode, TrackingTarget, PerformanceBenchmark string
	Meta                                           SourceMeta
}
type SeriesPoint struct {
	Date  time.Time
	Value float64
}
type NAVSeries struct {
	FundCode, Basis string
	Points          []SeriesPoint
	Meta            SourceMeta
}
type ETFQuote struct {
	FundCode, Name           string
	Price, IOPV, PremiumRate *float64
	Meta                     SourceMeta
}
type BenchmarkSeries struct {
	BenchmarkID, Basis string
	Points             []SeriesPoint
	Meta               SourceMeta
}
type FXRate struct {
	Currency, Unit string
	CNYPerUnit     float64
	Meta           SourceMeta
}

type FundCatalogProvider interface {
	DiscoverQDII(context.Context) ([]FundCatalogItem, error)
}
type PurchaseProvider interface {
	PurchaseLimits(context.Context) ([]PurchaseLimit, error)
}
type FundOverviewProvider interface {
	Overview(context.Context, string) (FundOverview, error)
}
type FundNAVProvider interface {
	FundNAV(context.Context, string) (NAVSeries, error)
}
type ETFProvider interface {
	ETFQuotes(context.Context) ([]ETFQuote, error)
}
type BenchmarkProvider interface {
	Benchmark(context.Context, string, time.Time, time.Time) (BenchmarkSeries, error)
}
type FXProvider interface {
	ReferenceRates(context.Context, time.Time) ([]FXRate, error)
}

type Availability string

const (
	Fresh       Availability = "fresh"
	Stale       Availability = "stale"
	Unavailable Availability = "unavailable"
)

func Classify(fetchedAt, now time.Time, freshFor time.Duration) Availability {
	if fetchedAt.IsZero() {
		return Unavailable
	}
	if now.Sub(fetchedAt) <= freshFor {
		return Fresh
	}
	return Stale
}
