package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type RelayProvider struct {
	BaseURL string
	Token   string
	Client  *http.Client
}

func NewRelayProvider(baseURL, token string) *RelayProvider {
	baseURL = strings.TrimRight(baseURL, "/")
	return &RelayProvider{
		BaseURL: baseURL,
		Token:   token,
		Client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *RelayProvider) Name() string      { return "中转站" }
func (p *RelayProvider) ShortLabel() string { return "R" }

type relayUserResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Quota     json.Number `json:"quota"`
		UsedQuota json.Number `json:"used_quota"`
		Username  string      `json:"username"`
	} `json:"data"`
}

func (p *RelayProvider) FetchStatus() (*UsageReport, error) {
	url := p.BaseURL + "/api/user/self"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.Token)
	req.Header.Set("User-Agent", "TokenTray/0.1")

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("网络错误: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("Token 无效（注意：需要 dashboard 的 access token，不是 sk- 开头的 API key）")
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body)[:min(len(body), 100)])
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed relayUserResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("解析失败: %w", err)
	}

	if !parsed.Success {
		return nil, fmt.Errorf("接口返回失败: %s", parsed.Message)
	}

	report := &UsageReport{
		ProviderName: p.Name(),
		ShortLabel:   p.ShortLabel(),
		LastUpdated:  time.Now(),
	}

	quota, _ := parsed.Data.Quota.Float64()
	used, _ := parsed.Data.UsedQuota.Float64()
	total := quota + used

	var pct float64
	if total > 0 {
		pct = (used / total) * 100
	}

	quotaInt := int64(quota)
	usedInt := int64(used)
	totalInt := int64(total)

	label := "中转站额度"
	if parsed.Data.Username != "" {
		label = fmt.Sprintf("中转站 · %s", parsed.Data.Username)
	}

	report.Windows = []QuotaWindow{{
		Label:      label,
		Used:       &usedInt,
		Limit:      &totalInt,
		Percentage: &pct,
	}}

	if quotaInt < 10000 {
		gone := "余额不足"
		report.Windows[0].Label = label + " ⚠ " + gone
	}

	return report, nil
}
