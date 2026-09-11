package marketdata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPurchaseDetailsDistinguishNoSalesFromZeroLimit(t *testing.T) {
	for _, code := range []string{"021778", "021000", "022525"} {
		page := `<title>基金(` + code + `)基金费率</title><p>申购状态 限大额 赎回状态 开放赎回</p><h4>申购与赎回金额</h4><p>该基金尚未开通天天基金代销，暂无相关数据。</p><h4>交易确认日</h4>`
		item, err := parsePurchaseDetail([]byte(page), code)
		if err != nil || item.Status != "not_sold" || item.DailyLimitCents != nil || item.MinimumCents != nil {
			t.Fatalf("%+v %v", item, err)
		}
	}
}

func TestPurchaseDetailPositiveAmountAndIdentity(t *testing.T) {
	page := `<title>基金(270042)基金费率</title><table><tr><td>申购状态</td><td>暂停大额申购</td><td>赎回状态</td><td>开放赎回</td></tr></table><h4>申购与赎回金额</h4><table><tr><td>申购起点</td><td>5.00元</td><td>日累计申购限额</td><td>5.00元</td></tr></table><h4>交易确认日</h4>`
	item, err := parsePurchaseDetail([]byte(page), "270042")
	if err != nil || item.Status != "limited" || item.DailyLimitCents == nil || *item.DailyLimitCents != 500 {
		t.Fatalf("%+v %v", item, err)
	}
	if _, err := parsePurchaseDetail([]byte(page), "006075"); err == nil {
		t.Fatal("accepted another fund's page")
	}
	if _, err := parsePurchaseDetail([]byte(`<title>(270042)</title><script>申购与赎回金额 该基金尚未开通天天基金代销，暂无相关数据。 交易确认日</script>`), "270042"); err == nil {
		t.Fatal("accepted script as data")
	}
}

func TestCollectorEnrichesOnlyMappedMissingLimits(t *testing.T) {
	requests := 0
	fail := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if fail {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`<title>基金(021000)基金费率</title><h4>申购与赎回金额</h4><p>该基金尚未开通天天基金代销，暂无相关数据。</p><h4>交易确认日</h4>`))
	}))
	defer upstream.Close()
	provider := NewEastMoneyPurchaseClient()
	provider.BaseURL = upstream.URL
	snapshots := &collectorSnapshotStore{}
	c := Collector{PurchaseDetails: provider, Snapshots: &SnapshotService{Store: snapshots}}
	positive := int64(500)
	items := []PurchaseLimit{{FundCode: "021000"}, {FundCode: "270042", DailyLimitCents: &positive}, {FundCode: "999999"}}
	if err := c.refreshPurchaseDetails(context.Background(), items, map[string]bool{"021000": true, "270042": true}); err != nil {
		t.Fatal(err)
	}
	if requests != 1 || len(snapshots.snapshots) != 1 {
		t.Fatalf("requests %d snapshots %d", requests, len(snapshots.snapshots))
	}
	var saved normalizedPurchase
	if err := json.Unmarshal(snapshots.snapshots[0].NormalizedJSON, &saved); err != nil || saved.Status != "not_sold" {
		t.Fatalf("%+v %v", saved, err)
	}
	fail = true
	if err := c.refreshPurchaseDetails(context.Background(), items, map[string]bool{"021000": true}); err != nil {
		t.Fatal("detail failure must keep bulk snapshot", err)
	}
	if len(snapshots.snapshots) != 1 {
		t.Fatal("failed response was stored")
	}
}
