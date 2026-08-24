package marketdata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type TushareClient struct {
	Token    string
	Endpoint string
	HTTP     *http.Client
	Now      func() time.Time
}

func NewTushareClient(token string, timeout time.Duration) *TushareClient {
	return &TushareClient{Token: token, Endpoint: "https://api.tushare.pro", HTTP: &http.Client{Timeout: timeout}, Now: time.Now}
}

type tushareRequest struct {
	APIName string         `json:"api_name"`
	Token   string         `json:"token"`
	Params  map[string]any `json:"params"`
	Fields  string         `json:"fields"`
}
type tushareResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Fields []string `json:"fields"`
		Items  [][]any  `json:"items"`
	} `json:"data"`
}

func (c *TushareClient) call(ctx context.Context, name string, params map[string]any, fields string) (tushareResponse, error) {
	if c.Token == "" {
		return tushareResponse{}, fmt.Errorf("tushare_not_configured")
	}
	body, _ := json.Marshal(tushareRequest{APIName: name, Token: c.Token, Params: params, Fields: fields})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return tushareResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return tushareResponse{}, fmt.Errorf("tushare request failed")
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return tushareResponse{}, fmt.Errorf("tushare read failed")
	}
	var result tushareResponse
	if json.Unmarshal(raw, &result) != nil || result.Code != 0 {
		return tushareResponse{}, fmt.Errorf("tushare upstream rejected request")
	}
	if len(result.Data.Fields) == 0 || len(result.Data.Items) == 0 {
		return tushareResponse{}, fmt.Errorf("tushare empty response")
	}
	return result, nil
}
func (c *TushareClient) FundNAV(ctx context.Context, code string) (NAVSeries, error) {
	r, err := c.call(ctx, "fund_nav", map[string]any{"ts_code": code}, "ts_code,nav_date,adj_nav")
	if err != nil {
		return NAVSeries{}, err
	}
	points, err := tusharePoints(r, "nav_date", "adj_nav")
	if err != nil {
		return NAVSeries{}, err
	}
	return NAVSeries{FundCode: code, Basis: "adjusted_nav", Points: points, Meta: SourceMeta{Provider: "tushare", SourceURL: "https://tushare.pro/document/2?doc_id=119", ObservedAt: points[len(points)-1].Date, FetchedAt: c.Now().UTC()}}, nil
}
func (c *TushareClient) Benchmark(ctx context.Context, id string, from, to time.Time) (BenchmarkSeries, error) {
	r, err := c.call(ctx, "index_global", map[string]any{"ts_code": id, "start_date": from.Format("20060102"), "end_date": to.Format("20060102")}, "ts_code,trade_date,close")
	if err != nil {
		return BenchmarkSeries{}, err
	}
	points, err := tusharePoints(r, "trade_date", "close")
	if err != nil {
		return BenchmarkSeries{}, err
	}
	return BenchmarkSeries{BenchmarkID: id, Basis: "price", Points: points, Meta: SourceMeta{Provider: "tushare", SourceURL: "https://tushare.pro/document/2?doc_id=211", ObservedAt: points[len(points)-1].Date, FetchedAt: c.Now().UTC()}}, nil
}
func (c *TushareClient) TradeCalendar(ctx context.Context, from, to time.Time) ([]time.Time, error) {
	r, err := c.call(ctx, "trade_cal", map[string]any{"start_date": from.Format("20060102"), "end_date": to.Format("20060102"), "is_open": "1"}, "cal_date,is_open")
	if err != nil {
		return nil, err
	}
	index := fieldIndex(r.Data.Fields, "cal_date")
	if index < 0 {
		return nil, fmt.Errorf("tushare trade calendar contract changed")
	}
	out := []time.Time{}
	for _, row := range r.Data.Items {
		if len(row) <= index {
			return nil, fmt.Errorf("tushare array contract changed")
		}
		d, err := parseDate(fmt.Sprint(row[index]))
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}
func (c *TushareClient) ReferenceRates(ctx context.Context, at time.Time) ([]FXRate, error) {
	r, err := c.call(ctx, "fx_daily", map[string]any{"trade_date": at.Format("20060102")}, "ts_code,trade_date,close")
	if err != nil {
		return nil, err
	}
	codeI, dateI, valueI := fieldIndex(r.Data.Fields, "ts_code"), fieldIndex(r.Data.Fields, "trade_date"), fieldIndex(r.Data.Fields, "close")
	if codeI < 0 || dateI < 0 || valueI < 0 {
		return nil, fmt.Errorf("tushare fx contract changed")
	}
	out := []FXRate{}
	for _, row := range r.Data.Items {
		if len(row) <= valueI {
			return nil, fmt.Errorf("tushare array contract changed")
		}
		value := parseFloatPtr(row[valueI])
		if value == nil {
			continue
		}
		date, _ := parseDate(fmt.Sprint(row[dateI]))
		out = append(out, FXRate{Currency: fmt.Sprint(row[codeI]), Unit: "provider_quote", CNYPerUnit: *value, Meta: SourceMeta{Provider: "tushare", SourceURL: "https://tushare.pro/document/2?doc_id=179", ObservedAt: date, FetchedAt: c.Now().UTC()}})
	}
	return out, nil
}
func fieldIndex(fields []string, want string) int {
	for i, v := range fields {
		if v == want {
			return i
		}
	}
	return -1
}
func tusharePoints(r tushareResponse, dateField, valueField string) ([]SeriesPoint, error) {
	di, vi := fieldIndex(r.Data.Fields, dateField), fieldIndex(r.Data.Fields, valueField)
	if di < 0 || vi < 0 {
		return nil, fmt.Errorf("tushare fields contract changed")
	}
	out := make([]SeriesPoint, 0, len(r.Data.Items))
	for _, row := range r.Data.Items {
		if len(row) <= di || len(row) <= vi {
			return nil, fmt.Errorf("tushare array contract changed")
		}
		date, err := parseDate(fmt.Sprint(row[di]))
		if err != nil {
			return nil, err
		}
		value := parseFloatPtr(row[vi])
		if value == nil {
			return nil, fmt.Errorf("tushare value unavailable")
		}
		out = append(out, SeriesPoint{Date: date, Value: *value})
	}
	return out, nil
}
