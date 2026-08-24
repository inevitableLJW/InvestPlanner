package marketdata

import (
	"math"
	"sort"
	"time"
)

type TrackingErrorResult struct {
	Value           float64   `json:"value"`
	Count           int       `json:"count"`
	Through         time.Time `json:"through"`
	Basis           string    `json:"basis"`
	BenchmarkSymbol string    `json:"benchmarkSymbol"`
}

func AnnualizedTrackingError(fund []SeriesPoint, fundBasis string, benchmark []SeriesPoint, benchmarkBasis string, now time.Time) (float64, int, time.Time, string) {
	strict := fundBasis == "adjusted_nav" && benchmarkBasis == "total_return"
	proxy := fundBasis == "cumulative_nav_proxy" && benchmarkBasis == "adjusted_close_cny_proxy"
	if !strict && !proxy {
		return 0, 0, time.Time{}, "收益口径不兼容"
	}
	fundReturns := seriesReturns(fund, now.AddDate(-1, 0, 0))
	baseReturns := seriesReturns(benchmark, now.AddDate(-1, 0, 0))
	diffs := []float64{}
	var through time.Time
	for date, fundReturn := range fundReturns {
		if baseReturn, ok := baseReturns[date]; ok {
			diffs = append(diffs, fundReturn-baseReturn)
			if date.After(through) {
				through = date
			}
		}
	}
	return annualizeTrackingDiffs(diffs, through, now)
}

// AnnualizedQDIIETFTrackingError aligns a mainland ETF close with the latest
// completed US session before that China trading date. This avoids comparing a
// China close with the later QQQ/SPY close from the same calendar date and also
// aggregates US returns across mainland holidays.
func AnnualizedQDIIETFTrackingError(fund []SeriesPoint, fundBasis string, benchmark []SeriesPoint, benchmarkBasis string, now time.Time) (float64, int, time.Time, string) {
	validFundBasis := fundBasis == "adjusted_close_cny" || fundBasis == "close_cny_no_corporate_action"
	if !validFundBasis || benchmarkBasis != "adjusted_close_cny_proxy" {
		return 0, 0, time.Time{}, "收益口径不兼容"
	}
	sort.Slice(fund, func(i, j int) bool { return fund[i].Date.Before(fund[j].Date) })
	sort.Slice(benchmark, func(i, j int) bool { return benchmark[i].Date.Before(benchmark[j].Date) })
	from := now.AddDate(-1, 0, 0)
	diffs := []float64{}
	var through time.Time
	for i := 1; i < len(fund); i++ {
		if fund[i].Date.Before(from) || fund[i-1].Value <= 0 {
			continue
		}
		previousBase, _, previousOK := latestPointBefore(benchmark, fund[i-1].Date)
		currentBase, currentDate, currentOK := latestPointBefore(benchmark, fund[i].Date)
		if !previousOK || !currentOK || previousBase <= 0 || currentDate.IsZero() {
			continue
		}
		fundReturn := fund[i].Value/fund[i-1].Value - 1
		baseReturn := currentBase/previousBase - 1
		diffs = append(diffs, fundReturn-baseReturn)
		if currentDate.After(through) {
			through = currentDate
		}
	}
	return annualizeTrackingDiffs(diffs, through, now)
}

func latestPointBefore(points []SeriesPoint, cutoff time.Time) (float64, time.Time, bool) {
	for i := len(points) - 1; i >= 0; i-- {
		if points[i].Date.Before(cutoff) {
			return points[i].Value, points[i].Date, true
		}
	}
	return 0, time.Time{}, false
}

func annualizeTrackingDiffs(diffs []float64, through, now time.Time) (float64, int, time.Time, string) {
	if len(diffs) < 120 {
		return 0, len(diffs), through, "共同观测日不足 120 个"
	}
	if now.Sub(through) > 14*24*time.Hour {
		return 0, len(diffs), through, "数据已陈旧"
	}
	mean := 0.0
	for _, value := range diffs {
		mean += value
	}
	mean /= float64(len(diffs))
	sum := 0.0
	for _, value := range diffs {
		difference := value - mean
		sum += difference * difference
	}
	return math.Sqrt(sum/float64(len(diffs)-1)) * math.Sqrt(252), len(diffs), through, ""
}

func seriesReturns(points []SeriesPoint, from time.Time) map[time.Time]float64 {
	sort.Slice(points, func(i, j int) bool { return points[i].Date.Before(points[j].Date) })
	out := map[time.Time]float64{}
	for i := 1; i < len(points); i++ {
		if points[i].Date.Before(from) || points[i-1].Value <= 0 {
			continue
		}
		out[points[i].Date] = points[i].Value/points[i-1].Value - 1
	}
	return out
}
