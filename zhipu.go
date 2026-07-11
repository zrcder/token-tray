package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

var zhipuEndpoints = []string{
	"https://open.bigmodel.cn/api/monitor/usage/quota/limit",
	"https://bigmodel.cn/api/monitor/usage/quota/limit",
}

type zhipuLimit struct {
	Type          string  `json:"type"`
	Usage         *int64  `json:"usage"`
	CurrentValue  *int64  `json:"currentValue"`
	Remaining     *int64  `json:"remaining"`
	Percentage    float64 `json:"percentage"`
	NextResetTime *int64  `json:"nextResetTime"`
}

type zhipuResponse struct {
	Success bool `json:"success"`
	Msg     string `json:"msg"`
	Data    struct {
		Level  string       `json:"level"`
		Limits []zhipuLimit `json:"limits"`
	} `json:"data"`
}

type ZhipuProvider struct {
	APIKey string
	Client *http.Client
}

func NewZhipuProvider(apiKey string) *ZhipuProvider {
	return &ZhipuProvider{
		APIKey: apiKey,
		Client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *ZhipuProvider) Name() string      { return "智谱 GLM" }
func (p *ZhipuProvider) ShortLabel() string { return "智" }

func (p *ZhipuProvider) FetchStatus() (*UsageReport, error) {
	resp, err := p.fetchWithFallback()
	if err != nil {
		return nil, err
	}

	report := &UsageReport{
		ProviderName: p.Name(),
		ShortLabel:   p.ShortLabel(),
		PlanLevel:    resp.Data.Level,
		LastUpdated:  time.Now(),
	}

	var tokenLimits []zhipuLimit
	for _, l := range resp.Data.Limits {
		if l.Type == "TOKENS_LIMIT" {
			tokenLimits = append(tokenLimits, l)
		}
	}
	sort.Slice(tokenLimits, func(i, j int) bool {
		a := int64(0)
		b := int64(0)
		if tokenLimits[i].NextResetTime != nil {
			a = *tokenLimits[i].NextResetTime
		}
		if tokenLimits[j].NextResetTime != nil {
			b = *tokenLimits[j].NextResetTime
		}
		return a < b
	})

	if len(tokenLimits) >= 1 {
		t := tokenLimits[0]
		pct := t.Percentage
		report.Windows = append(report.Windows, QuotaWindow{
			Label:       "5 小时 Token",
			Used:        t.CurrentValue,
			Limit:       t.Usage,
			Percentage:  &pct,
			NextResetMs: t.NextResetTime,
		})
	}
	if len(tokenLimits) >= 2 {
		t := tokenLimits[1]
		pct := t.Percentage
		report.Windows = append(report.Windows, QuotaWindow{
			Label:       "每周 Token",
			Used:        t.CurrentValue,
			Limit:       t.Usage,
			Percentage:  &pct,
			NextResetMs: t.NextResetTime,
		})
	}

	for _, l := range resp.Data.Limits {
		if l.Type == "TIME_LIMIT" {
			pct := l.Percentage
			report.Windows = append(report.Windows, QuotaWindow{
				Label:       "MCP 月度",
				Used:        l.CurrentValue,
				Limit:       l.Usage,
				Percentage:  &pct,
				NextResetMs: l.NextResetTime,
			})
			break
		}
	}

	return report, nil
}

func (p *ZhipuProvider) fetchWithFallback() (*zhipuResponse, error) {
	var lastErr error
	for _, url := range zhipuEndpoints {
		r, err := p.fetchOne(url)
		if err == nil {
			return r, nil
		}
		lastErr = err
		// 401/403 — auth failure is same on both endpoints; bail.
		if httpErr, ok := err.(*httpStatusError); ok && (httpErr.code == 401 || httpErr.code == 403) {
			return nil, fmt.Errorf("API Key 无效或已过期 (HTTP %d)", httpErr.code)
		}
	}
	return nil, fmt.Errorf("所有端点请求失败: %w", lastErr)
}

type httpStatusError struct {
	code int
	body string
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.code, e.body)
}

func (p *ZhipuProvider) fetchOne(url string) (*zhipuResponse, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	// Zhipu monitor API: raw key, NO "Bearer " prefix.
	req.Header.Set("Authorization", p.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TokenTray/0.1")

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, &httpStatusError{code: resp.StatusCode, body: string(body)[:min(200, len(body))]}
	}
	if resp.StatusCode != 200 {
		return nil, &httpStatusError{code: resp.StatusCode, body: string(body)}
	}

	var parsed zhipuResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("响应解析失败: %w", err)
	}
	if !parsed.Success {
		return nil, fmt.Errorf("接口返回失败: %s", parsed.Msg)
	}
	return &parsed, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
