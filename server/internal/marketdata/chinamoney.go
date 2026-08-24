package marketdata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const ChinaMoneySource = "https://www.chinamoney.com.cn/chinese/bkccpr/index.html?tab=2"

type ChinaMoneyClient struct {
	Endpoint string
	HTTP     *http.Client
	Now      func() time.Time
}

func NewChinaMoneyClient(endpoint string, timeout time.Duration) *ChinaMoneyClient {
	return &ChinaMoneyClient{Endpoint: endpoint, HTTP: &http.Client{Timeout: timeout}, Now: time.Now}
}

type chinaMoneyRow struct {
	Currency string  `json:"currency"`
	Date     string  `json:"date"`
	Value    float64 `json:"value"`
	Unit     string  `json:"unit"`
}

func (c *ChinaMoneyClient) ReferenceRates(ctx context.Context, at time.Time) ([]FXRate, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chinamoney request failed")
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("chinamoney read failed")
	}
	rows, err := parseChinaMoney(raw)
	if err != nil {
		return nil, err
	}
	result := []FXRate{}
	for _, row := range rows {
		if row.Currency != "USD" && row.Currency != "HKD" {
			continue
		}
		date, err := parseDate(row.Date)
		if err != nil {
			return nil, fmt.Errorf("chinamoney date contract changed")
		}
		divisor := 1.0
		if strings.Contains(row.Unit, "100") || row.Unit == "CNY/100" {
			divisor = 100
		}
		result = append(result, FXRate{Currency: row.Currency, Unit: row.Unit, CNYPerUnit: row.Value / divisor, Meta: SourceMeta{Provider: "chinamoney", SourceURL: ChinaMoneySource, ObservedAt: date, FetchedAt: c.Now().UTC()}})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("chinamoney empty response")
	}
	return result, nil
}
func parseChinaMoney(raw []byte) ([]chinaMoneyRow, error) {
	trimmed := strings.TrimSpace(string(raw))
	if i := strings.Index(trimmed, "("); i >= 0 && strings.HasSuffix(trimmed, ")") {
		trimmed = trimmed[i+1 : len(trimmed)-1]
	}
	var envelope struct {
		Records []chinaMoneyRow `json:"records"`
	}
	if json.Unmarshal([]byte(trimmed), &envelope) != nil || len(envelope.Records) == 0 {
		return nil, fmt.Errorf("chinamoney response contract changed")
	}
	for _, row := range envelope.Records {
		if row.Currency == "" || row.Date == "" || row.Unit == "" || row.Value <= 0 {
			return nil, fmt.Errorf("chinamoney field contract changed")
		}
	}
	return envelope.Records, nil
}
