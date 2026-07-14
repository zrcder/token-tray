package main

import (
	"fmt"
	"os"
	"time"
)

type mockProvider struct {
	name       string
	shortLabel string
	windows    []QuotaWindow
	errMsg     string
}

func (m *mockProvider) Name() string       { return m.name }
func (m *mockProvider) ShortLabel() string { return m.shortLabel }
func (m *mockProvider) FetchStatus() (*UsageReport, error) {
	if m.errMsg != "" {
		return nil, fmt.Errorf("%s", m.errMsg)
	}
	return &UsageReport{
		ProviderName: m.name,
		ShortLabel:   m.shortLabel,
		Windows:      m.windows,
		LastUpdated:  time.Now(),
	}, nil
}

type testScenario struct {
	name      string
	providers []Provider
}

var (
	testScenarioIdx int
	testPaused      bool
)

func testScenarios() []testScenario {
	now := time.Now()
	h1 := now.Add(4 * time.Hour).UnixMilli()
	h5 := now.Add(12 * time.Hour).UnixMilli()
	d3 := now.Add(3 * time.Hour).UnixMilli()
	w5 := now.Add(5 * 24 * time.Hour).UnixMilli()

	pct := func(v float64) *float64 { return &v }
	reset := func(v int64) *int64 { return &v }
	zhipu := func(windows ...QuotaWindow) Provider {
		return &mockProvider{name: "智谱 GLM (测试)", shortLabel: "智谱", windows: windows}
	}
	ds := func(p float64) Provider {
		return &mockProvider{name: "DeepSeek (测试)", shortLabel: "DS", windows: []QuotaWindow{
			{Label: "余额", Percentage: pct(p), NextResetMs: reset(h5)},
		}}
	}

	return []testScenario{
		{
			name: "🟢 正常 10%",
			providers: []Provider{
				zhipu(QuotaWindow{Label: "时度", Percentage: pct(10), NextResetMs: reset(h1)}),
			},
		},
		{
			name: "🟡 注意 35%",
			providers: []Provider{
				zhipu(QuotaWindow{Label: "时度", Percentage: pct(35), NextResetMs: reset(h1)}),
			},
		},
		{
			name: "🟠 警告 60%",
			providers: []Provider{
				zhipu(QuotaWindow{Label: "时度", Percentage: pct(60), NextResetMs: reset(h1)}),
			},
		},
		{
			name: "🔴 危险 85%",
			providers: []Provider{
				zhipu(QuotaWindow{Label: "时度", Percentage: pct(85), NextResetMs: reset(h1)}),
			},
		},
		{
			name: "⬜ 无数据",
			providers: []Provider{
				zhipu(QuotaWindow{Label: "时度", Percentage: nil, NextResetMs: nil}),
			},
		},
		{
			name: "📊 多窗口",
			providers: []Provider{
				zhipu(
					QuotaWindow{Label: "时度", Percentage: pct(18), NextResetMs: reset(h1)},
					QuotaWindow{Label: "周度", Percentage: pct(42), NextResetMs: reset(w5)},
					QuotaWindow{Label: "月度", Percentage: pct(68), NextResetMs: reset(d3)},
				),
			},
		},
		{
			name: "📱 多 Provider",
			providers: []Provider{
				zhipu(QuotaWindow{Label: "时度", Percentage: pct(30), NextResetMs: reset(h1)}),
				ds(88),
			},
		},
		{
			name: "❌ API 错误",
			providers: []Provider{
				&mockProvider{name: "智谱 GLM (测试)", shortLabel: "智谱", errMsg: "401 Unauthorized: API Key 无效"},
			},
		},
	}
}

func currentTestProviders() []Provider {
	scenarios := testScenarios()
	s := scenarios[testScenarioIdx%len(scenarios)]
	out := make([]Provider, len(s.providers))
	copy(out, s.providers)
	return out
}

func currentTestScenarioName() string {
	scenarios := testScenarios()
	idx := testScenarioIdx % len(scenarios)
	total := len(scenarios)
	return fmt.Sprintf("%d/%d  %s", idx+1, total, scenarios[idx].name)
}

func advanceTestScenario() {
	testScenarioIdx++
}

func generateTestIcons() {
	_ = os.MkdirAll("screenshots", 0755)

	states := []struct {
		name string
		frac *float64
	}{
		{"green", ptr(0.15)},
		{"yellow", ptr(0.35)},
		{"orange", ptr(0.65)},
		{"red", ptr(0.85)},
		{"gray", nil},
	}
	for _, s := range states {
		data := generateSegmentedIcon([]DotColor{colorForFraction(s.frac)})
		_ = os.WriteFile("screenshots/icon-test-"+s.name+".png", data, 0644)
	}

	combos := []struct {
		name  string
		fracs []float64
	}{
		{"green-yellow", []float64{0.10, 0.40}},
		{"orange-red", []float64{0.60, 0.80}},
		{"green-yellow-orange", []float64{0.10, 0.40, 0.60}},
		{"mixed", []float64{0.05, 0.95}},
	}
	for _, c := range combos {
		var segs []DotColor
		for _, f := range c.fracs {
			segs = append(segs, colorForFraction(ptr(f)))
		}
		data := generateSegmentedIcon(segs)
		_ = os.WriteFile("screenshots/icon-test-"+c.name+".png", data, 0644)
	}
}

func ptr(v float64) *float64 { return &v }
