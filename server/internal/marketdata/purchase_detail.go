package marketdata

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// Purchase details disambiguate the bulk list's zero sentinel. This is only
// consulted by the collector, never by a user's business request.
type PurchaseDetailProvider interface {
	PurchaseDetail(context.Context, string) (PurchaseLimit, error)
}
type EastMoneyPurchaseClient struct {
	HTTP    *http.Client
	BaseURL string
	Now     func() time.Time
}

func NewEastMoneyPurchaseClient() *EastMoneyPurchaseClient {
	return &EastMoneyPurchaseClient{HTTP: &http.Client{Timeout: 5 * time.Second}, BaseURL: "https://fundf10.eastmoney.com", Now: time.Now}
}

var fundCodePattern = regexp.MustCompile(`^[0-9]{6}$`)

func (c *EastMoneyPurchaseClient) PurchaseDetail(ctx context.Context, code string) (PurchaseLimit, error) {
	if !fundCodePattern.MatchString(code) {
		return PurchaseLimit{}, fmt.Errorf("invalid fund code")
	}
	source := c.BaseURL + "/jjfl_" + code + ".html"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return PurchaseLimit{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return PurchaseLimit{}, fmt.Errorf("purchase detail unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return PurchaseLimit{}, fmt.Errorf("purchase detail status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return PurchaseLimit{}, fmt.Errorf("purchase detail read failed")
	}
	result, err := parsePurchaseDetail(body, code)
	if err != nil {
		return result, err
	}
	now := c.Now().UTC()
	result.Meta = SourceMeta{Provider: "eastmoney_detail", SourceURL: source, ObservedAt: now, FetchedAt: now}
	return result, nil
}

func parsePurchaseDetail(body []byte, code string) (PurchaseLimit, error) {
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return PurchaseLimit{}, fmt.Errorf("invalid purchase detail")
	}
	var text strings.Builder
	var title strings.Builder
	var visit func(*html.Node, bool)
	visit = func(n *html.Node, inTitle bool) {
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return
		}
		inTitle = inTitle || n.Type == html.ElementNode && n.Data == "title"
		if n.Type == html.TextNode {
			text.WriteString(n.Data)
			text.WriteByte(' ')
			if inTitle {
				title.WriteString(n.Data)
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			visit(child, inTitle)
		}
	}
	visit(doc, false)
	if !strings.Contains(title.String(), "("+code+")") {
		return PurchaseLimit{}, fmt.Errorf("purchase detail identity mismatch")
	}
	plain := strings.Join(strings.Fields(text.String()), " ")
	start := strings.Index(plain, "申购与赎回金额")
	if start < 0 {
		return PurchaseLimit{}, fmt.Errorf("purchase detail contract changed")
	}
	end := strings.Index(plain[start:], "交易确认日")
	if end < 0 {
		return PurchaseLimit{}, fmt.Errorf("purchase detail contract changed")
	}
	amounts := plain[start : start+end]
	result := PurchaseLimit{FundCode: code, Venue: "otc", Channel: "天天基金"}
	if strings.Contains(amounts, "该基金尚未开通天天基金代销，暂无相关数据") {
		result.Status = "not_sold"
		result.RawStatus = "该渠道未开通代销"
		return result, nil
	}
	status := regexp.MustCompile(`申购状态\s+(\S+)\s+赎回状态`).FindStringSubmatch(plain)
	if len(status) != 2 {
		return result, fmt.Errorf("purchase status contract changed")
	}
	result.RawStatus = status[1]
	result.Status = normalizePurchaseStatus(status[1])
	amount := func(label string) *int64 {
		match := regexp.MustCompile(regexp.QuoteMeta(label) + `\s+([0-9,]+(?:\.[0-9]+)?)\s*元`).FindStringSubmatch(amounts)
		if len(match) != 2 {
			return nil
		}
		return parseCNYCents(match[1])
	}
	result.MinimumCents = amount("申购起点")
	result.DailyLimitCents = amount("日累计申购限额")
	if result.DailyLimitCents != nil && (*result.DailyLimitCents <= 0 || result.MinimumCents != nil && *result.DailyLimitCents < *result.MinimumCents) {
		result.DailyLimitCents = nil
	}
	return result, nil
}
