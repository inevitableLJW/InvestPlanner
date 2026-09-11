package service

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"investplanner/server/internal/database"
	"investplanner/server/internal/marketdata"
)

func TestTrackingErrorRequiresAdjustedCompatibleAndEnoughData(t *testing.T) {
	now := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	fund := make([]marketdata.SeriesPoint, 0, 150)
	benchmark := make([]marketdata.SeriesPoint, 0, 150)
	for i := 0; i < 150; i++ {
		date := now.AddDate(0, 0, -149+i)
		base := 100 * math.Pow(1.0005, float64(i))
		fund = append(fund, marketdata.SeriesPoint{Date: date, Value: base * (1 + 0.0001*math.Sin(float64(i)))})
		benchmark = append(benchmark, marketdata.SeriesPoint{Date: date, Value: base})
	}
	value, count, _, reason := TrackingError(fund, "adjusted_nav", benchmark, "total_return", now)
	if reason != "" || count < 120 || value <= 0 {
		t.Fatalf("value=%f count=%d reason=%s", value, count, reason)
	}
	if _, _, _, reason := TrackingError(fund, "unit_nav", benchmark, "price", now); reason != "收益口径不兼容" {
		t.Fatal(reason)
	}
	if value, count, _, reason := TrackingError(fund, "cumulative_nav_proxy", benchmark, "adjusted_close_cny_proxy", now); reason != "" || count < 120 || value <= 0 {
		t.Fatalf("proxy value=%f count=%d reason=%s", value, count, reason)
	}
	if _, _, _, reason := TrackingError(fund, "adjusted_close_cny", benchmark, "adjusted_close_cny_proxy", now); reason != "收益口径不兼容" {
		t.Fatalf("generic alignment must reject ETF basis, reason=%s", reason)
	}
	if _, _, _, reason := TrackingError(fund[:50], "adjusted_nav", benchmark[:50], "total_return", now); reason != "共同观测日不足 120 个" {
		t.Fatal(reason)
	}
}

func TestQDIIETFTrackingErrorUsesPreviousCompletedUSSession(t *testing.T) {
	now := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	benchmark := make([]marketdata.SeriesPoint, 0, 151)
	fund := make([]marketdata.SeriesPoint, 0, 150)
	for i := 0; i < 151; i++ {
		date := now.AddDate(0, 0, -151+i)
		base := 100 * math.Pow(1.0005, float64(i))
		benchmark = append(benchmark, marketdata.SeriesPoint{Date: date, Value: base})
		if i > 0 {
			fund = append(fund, marketdata.SeriesPoint{Date: date.AddDate(0, 0, 1), Value: base * (1 + 0.0001*math.Sin(float64(i)))})
		}
	}
	value, count, through, reason := marketdata.AnnualizedQDIIETFTrackingError(fund, "adjusted_close_cny", benchmark, "adjusted_close_cny_proxy", now)
	if reason != "" || count < 120 || value <= 0 || value > 0.01 || !through.Before(now) {
		t.Fatalf("value=%f count=%d through=%s reason=%s", value, count, through, reason)
	}
}

func TestCandidateOrderingKeepsUnknownDistinctFromZero(t *testing.T) {
	limit := int64(10000)
	known := QDIICandidate{FundCode: "A", PurchaseStatus: "open", DailyLimitCents: knownValue(limit, "10000", "p", "s", time.Now(), "fresh")}
	unknown := QDIICandidate{FundCode: "B", PurchaseStatus: "unknown", DailyLimitCents: unknownValue[int64]("unknown")}
	suspended := QDIICandidate{FundCode: "C", PurchaseStatus: "suspended", ExecutableCapacityCents: knownValue(int64(0), "0", "p", "s", time.Now(), "fresh")}
	if !betterOTC(known, unknown) || !betterOTC(unknown, suspended) {
		t.Fatal("unexpected ranking")
	}
	if suspended.ExecutableCapacityCents.Value == nil || *suspended.ExecutableCapacityCents.Value != 0 {
		t.Fatal("suspended capacity must be real zero")
	}
	if unknown.DailyLimitCents.Value != nil || unknown.DailyLimitCents.Display != "--" {
		t.Fatal("unknown must remain missing")
	}
}

func TestETFRequiresIOPVAndChoosesLowestAbsolutePremium(t *testing.T) {
	a, b := 0.03, -0.01
	first := QDIICandidate{PremiumRate: knownValue(a, "3%", "p", "s", time.Now(), "fresh")}
	second := QDIICandidate{PremiumRate: knownValue(b, "-1%", "p", "s", time.Now(), "fresh")}
	if !betterETF(second, first) {
		t.Fatal("expected lower absolute premium")
	}
	missing := QDIICandidate{PremiumRate: unknownValue[float64]("missing")}
	if betterETF(missing, first) {
		t.Fatal("missing IOPV/premium cannot win")
	}
}

func TestQDIIConfirmationValidatesAndNormalizesCurrentCandidates(t *testing.T) {
	result := QDIIResult{
		OTC:  []QDIICandidate{{FundCode: "000001", PurchaseStatus: "open"}, {FundCode: "000002", PurchaseStatus: "suspended"}},
		ETFs: []QDIICandidate{{FundCode: "159501"}},
	}
	otc, etf, err := validateQDIISelection(result, []string{" 000001 ", "000001"}, "159501")
	if err != nil || len(otc) != 1 || otc[0] != "000001" || etf != "159501" {
		t.Fatalf("otc=%v etf=%q err=%v", otc, etf, err)
	}
	if _, _, err := validateQDIISelection(result, []string{"000002"}, ""); err == nil {
		t.Fatal("suspended OTC fund must not be confirmable")
	}
	if _, _, err := validateQDIISelection(result, []string{"999999"}, ""); err == nil {
		t.Fatal("unmapped OTC fund must not be confirmable")
	}
	if _, _, err := validateQDIISelection(result, nil, "999999"); err == nil {
		t.Fatal("unmapped ETF must not be confirmable")
	}
}

func TestCalculateSelectedETFAmountUsesConfirmedOTCCapacity(t *testing.T) {
	firstCapacity := int64(12000)
	secondCapacity := int64(8000)
	result := QDIIResult{OTC: []QDIICandidate{
		{FundCode: "000001", ExecutableCapacityCents: AssistantValue[int64]{Value: &firstCapacity}},
		{FundCode: "000002", ExecutableCapacityCents: AssistantValue[int64]{Value: &secondCapacity}},
	}}
	amount := calculateSelectedETFAmount(result, []string{"000001", "000002"}, "159501", 100000)
	if amount == nil || *amount != 80000 {
		t.Fatalf("amount=%v, want 80000", amount)
	}
	if unknown := calculateSelectedETFAmount(QDIIResult{OTC: []QDIICandidate{{FundCode: "000001"}}}, []string{"000001"}, "159501", 100000); unknown != nil {
		t.Fatalf("unknown capacity must keep ETF amount unknown: %v", *unknown)
	}
	if withoutETF := calculateSelectedETFAmount(result, []string{"000001"}, "", 100000); withoutETF != nil {
		t.Fatalf("confirmation without ETF must not have an ETF amount: %v", *withoutETF)
	}
}

func TestBuildSelectedOTCPlansKeepsDailyAndMonthlyAmounts(t *testing.T) {
	daily := int64(500)
	total := int64(9000)
	result := QDIIResult{OTC: []QDIICandidate{
		{FundCode: "000001", DailyLimitCents: AssistantValue[int64]{Value: &daily}, ExecutableCapacityCents: AssistantValue[int64]{Value: &total}},
		{FundCode: "000002"},
	}}
	plans := buildSelectedOTCPlans(result, []string{"000001", "000002"})
	if len(plans) != 2 || plans[0].DailyAmountCents == nil || *plans[0].DailyAmountCents != daily || plans[0].TotalAmountCents == nil || *plans[0].TotalAmountCents != total {
		t.Fatalf("plans=%#v", plans)
	}
	if plans[1].DailyAmountCents != nil || plans[1].TotalAmountCents != nil {
		t.Fatalf("unknown amounts must remain nil: %#v", plans[1])
	}
}

func TestBuildQDIIConfirmationSummariesIncludesPersistedFundNames(t *testing.T) {
	selectedOTC, err := json.Marshal([]string{"270042", "006479"})
	if err != nil {
		t.Fatal(err)
	}
	confirmedAt := time.Date(2026, 8, 27, 1, 0, 0, 0, time.UTC)
	etfAmount := int64(77500)
	dailyA, totalA := int64(500), int64(10000)
	dailyC, totalC := int64(500), int64(12500)
	selectedOTCPlans, err := json.Marshal([]QDIIConfirmedOTCPlan{
		{FundCode: "270042", DailyAmountCents: &dailyA, TotalAmountCents: &totalA},
		{FundCode: "006479", DailyAmountCents: &dailyC, TotalAmountCents: &totalC},
	})
	if err != nil {
		t.Fatal(err)
	}
	summaries, err := buildQDIIConfirmationSummaries(
		[]database.QDIIAssistantConfirmation{{DestinationID: "q1", SelectedOTCFundCodesJSON: selectedOTC, SelectedOTCPlansJSON: selectedOTCPlans, SelectedETFFundCode: "513100", SelectedETFAmountCents: &etfAmount, ConfirmedAt: confirmedAt}},
		[]database.FundShare{
			{FundCode: "270042", Name: "广发纳斯达克100ETF联接人民币(QDII)A", ShareClass: "A"},
			{FundCode: "006479", Name: "广发纳斯达克100ETF联接人民币(QDII)C", ShareClass: "C"},
			{FundCode: "513100", Name: "纳指ETF国泰", ShareClass: "ETF"},
		},
	)
	if err != nil || len(summaries) != 1 {
		t.Fatalf("summaries=%#v err=%v", summaries, err)
	}
	summary := summaries[0]
	if summary.DestinationID != "q1" || len(summary.SelectedOTCFunds) != 2 || summary.SelectedOTCFunds[0].Name == "" || summary.SelectedOTCFunds[0].DailyAmountCents == nil || *summary.SelectedOTCFunds[0].DailyAmountCents != dailyA || summary.SelectedOTCFunds[1].TotalAmountCents == nil || *summary.SelectedOTCFunds[1].TotalAmountCents != totalC || summary.SelectedETFFund == nil || summary.SelectedETFFund.Name != "纳指ETF国泰" || summary.SelectedETFAmountCents == nil || *summary.SelectedETFAmountCents != etfAmount || !summary.ConfirmedAt.Equal(confirmedAt) {
		t.Fatalf("summary=%#v", summary)
	}
}
