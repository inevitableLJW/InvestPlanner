package marketdata

import (
	"math"
	"sort"
	"time"
)

type TrackingErrorResult struct {
	MethodVersion   int       `json:"methodVersion"`
	Value           float64   `json:"value"`
	Count           int       `json:"count"`
	Through         time.Time `json:"through"`
	Basis           string    `json:"basis"`
	BenchmarkSymbol string    `json:"benchmarkSymbol"`
}

const TrackingMethodVersion = 4

func trackingWindow(points []SeriesPoint, now time.Time) []SeriesPoint {
	from := now.AddDate(-1, 0, -30)
	window := make([]SeriesPoint, 0, len(points))
	for _, point := range points {
		if point.Date.IsZero() || !point.Date.Before(from) {
			window = append(window, point)
		}
	}
	return window
}

// Return observations must cover the same start AND end date. Comparing a
// multi-day fund return to a one-day benchmark return creates false tracking error.
type returnInterval struct{ From, To time.Time }

func validSeries(points []SeriesPoint, now time.Time) bool {
	seen := map[time.Time]bool{}
	for _, point := range points {
		if point.Date.IsZero() || point.Date.After(now) || seen[point.Date] || point.Value <= 0 || math.IsNaN(point.Value) || math.IsInf(point.Value, 0) {
			return false
		}
		seen[point.Date] = true
	}
	return true
}

func AnnualizedTrackingError(fund []SeriesPoint, fundBasis string, benchmark []SeriesPoint, benchmarkBasis string, now time.Time) (float64, int, time.Time, string) {
	strict := fundBasis == "adjusted_nav" && benchmarkBasis == "total_return"
	proxy := fundBasis == "cumulative_nav_proxy" && benchmarkBasis == "adjusted_close_cny_proxy"
	if !strict && !proxy {
		return 0, 0, time.Time{}, "收益口径不兼容"
	}
	fund, benchmark = trackingWindow(fund, now), trackingWindow(benchmark, now)
	if !validSeries(fund, now) || !validSeries(benchmark, now) {
		return 0, 0, time.Time{}, "收益序列包含无效数值、重复或未来日期"
	}
	fundReturns := seriesReturns(fund, now.AddDate(-1, 0, 0))
	baseReturns := seriesReturns(benchmark, now.AddDate(-1, 0, 0))
	diffs := []float64{}
	var through time.Time
	for date, fundReturn := range fundReturns {
		if baseReturn, ok := baseReturns[date]; ok {
			diffs = append(diffs, fundReturn-baseReturn)
			if date.To.After(through) {
				through = date.To
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
	fund, benchmark = trackingWindow(fund, now), trackingWindow(benchmark, now)
	if !validSeries(fund, now) || !validSeries(benchmark, now) {
		return 0, 0, time.Time{}, "收益序列包含无效数值、重复或未来日期"
	}
	fund = append([]SeriesPoint(nil), fund...)
	benchmark = append([]SeriesPoint(nil), benchmark...)
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
		if fund[i].Date.Sub(currentDate) > 7*24*time.Hour {
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
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return 0, len(diffs), through, "收益序列包含无效数值"
		}
		mean += value
	}
	mean /= float64(len(diffs))
	sum := 0.0
	for _, value := range diffs {
		difference := value - mean
		sum += difference * difference
	}
	value := math.Sqrt(sum/float64(len(diffs)-1)) * math.Sqrt(252)
	if math.IsInf(value, 0) || math.IsNaN(value) {
		return 0, len(diffs), through, "收益数值超出有效范围"
	}
	return value, len(diffs), through, ""
}

func seriesReturns(points []SeriesPoint, from time.Time) map[returnInterval]float64 {
	points = append([]SeriesPoint(nil), points...)
	sort.Slice(points, func(i, j int) bool { return points[i].Date.Before(points[j].Date) })
	out := map[returnInterval]float64{}
	for i := 1; i < len(points); i++ {
		if points[i].Date.Before(from) || points[i-1].Value <= 0 {
			continue
		}
		out[returnInterval{points[i-1].Date, points[i].Date}] = points[i].Value/points[i-1].Value - 1
	}
	return out
}
