package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type DeepSeekProvider struct {
	APIKey string
	Client *http.Client
}

func NewDeepSeekProvider(apiKey string) *DeepSeekProvider {
	return &DeepSeekProvider{
		APIKey: apiKey,
		Client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *DeepSeekProvider) Name() string      { return "DeepSeek" }
func (p *DeepSeekProvider) ShortLabel() string { return "DS" }

type deepSeekBalanceResponse struct {
	IsAvailable bool `json:"is_available"`
	BalanceInfos []struct {
		Currency       string `json:"currency"`
		TotalBalance   string `json:"total_balance"`
		GrantedBalance string `json:"granted_balance"`
		ToppedUpBalance string `json:"topped_up_balance"`
	} `json:"balance_infos"`
}

func (p *DeepSeekProvider) FetchStatus() (*UsageReport, error) {
	req, err := http.NewRequest("GET", "https://api.deepseek.com/user/balance", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("User-Agent", "TokenTray/0.1")

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("网络错误: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("DeepSeek API Key 无效")
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body)[:min(len(body), 100)])
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed deepSeekBalanceResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("解析失败: %w", err)
	}

	report := &UsageReport{
		ProviderName: p.Name(),
		ShortLabel:   p.ShortLabel(),
		LastUpdated:  time.Now(),
	}

	if !parsed.IsAvailable || len(parsed.BalanceInfos) == 0 {
		report.Error = "余额信息不可用"
		return report, nil
	}

	info := parsed.BalanceInfos[0]
	balance, _ := strconv.ParseFloat(info.TotalBalance, 64)

	pct := 0.0
	if balance < 1 {
		pct = 95
	} else if balance < 5 {
		pct = 80
	} else if balance < 10 {
		pct = 50
	} else {
		pct = 10
	}

	report.Windows = []QuotaWindow{{
		Label:      "余额",
		Percentage: &pct,
	}}
	report.PlanLevel = fmt.Sprintf("¥%s", info.TotalBalance)

	return report, nil
}
