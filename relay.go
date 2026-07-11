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
	BaseURL  string
	Token    string
	UserID   string
	Client   *http.Client
}

func NewRelayProvider(baseURL, token, userID string) *RelayProvider {
	baseURL = strings.TrimRight(baseURL, "/")
	return &RelayProvider{
		BaseURL: baseURL,
		Token:   token,
		UserID:  strings.TrimSpace(userID),
		Client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *RelayProvider) Name() string {
	host := p.BaseURL
	if idx := strings.Index(p.BaseURL, "://"); idx >= 0 {
		host = p.BaseURL[idx+3:]
	}
	return "中转站 " + host
}

func (p *RelayProvider) ShortLabel() string { return "R" }

type relayUserResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		ID        json.Number `json:"id"`
		Quota     json.Number `json:"quota"`
		UsedQuota json.Number `json:"used_quota"`
		Username  string      `json:"username"`
		Group     string      `json:"group"`
	} `json:"data"`
}

func (p *RelayProvider) FetchStatus() (*UsageReport, error) {
	authVariants := []string{
		"Bearer " + p.Token,
		p.Token,
	}

	var lastErr error
	for _, auth := range authVariants {
		r, err := p.tryFetch(auth)
		if err == nil {
			return r, nil
		}
		lastErr = err
		if !isAuthError(err) {
			return nil, err
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf(
			"认证失败。请确认:\n"+
				"1. Token 是 Dashboard 的「系统访问令牌」，不是 sk- API Key\n"+
				"2. 在中转站 → 个人设置 → 生成系统访问令牌\n"+
				"3. 原始错误: %v", lastErr,
		)
	}
	return nil, fmt.Errorf("未知错误")
}

func isAuthError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "401") || strings.Contains(msg, "认证") || strings.Contains(msg, "auth")
}

func (p *RelayProvider) tryFetch(authHeader string) (*UsageReport, error) {
	url := p.BaseURL + "/api/user/self"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("User-Agent", "TokenTray/0.1")
	if p.UserID != "" {
		req.Header.Set("New-Api-User", p.UserID)
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("网络错误: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("HTTP 401")
	}
	if resp.StatusCode != 200 {
		snippet := string(body)
		if len(snippet) > 100 {
			snippet = snippet[:100]
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, snippet)
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

	balanceYuan := quota / 500000.0
	totalYuan := total / 500000.0

	var label string
	if parsed.Data.Username != "" {
		label = fmt.Sprintf("%s", parsed.Data.Username)
	} else {
		label = "额度"
	}

	balanceStr := fmt.Sprintf("%.2f", balanceYuan)
	totalStr := fmt.Sprintf("%.2f", totalYuan)
	label = fmt.Sprintf("余额 ¥%s / ¥%s", balanceStr, totalStr)

	quotaInt := int64(quota)
	totalInt := int64(total)

	report.Windows = []QuotaWindow{{
		Label:      label,
		Used:       &quotaInt,
		Limit:      &totalInt,
		Percentage: &pct,
	}}

	if balanceYuan < 0.5 {
		report.Windows[0].Label = "⚠️ " + label + " (不足)"
	}

	return report, nil
}
