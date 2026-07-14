package main

import (
	"os"
	"time"
)

type mockProvider struct {
	name       string
	shortLabel string
	windows    []QuotaWindow
}

func (m *mockProvider) Name() string       { return m.name }
func (m *mockProvider) ShortLabel() string { return m.shortLabel }
func (m *mockProvider) FetchStatus() (*UsageReport, error) {
	return &UsageReport{
		ProviderName: m.name,
		ShortLabel:   m.shortLabel,
		Windows:      m.windows,
		LastUpdated:  time.Now(),
	}, nil
}

func makeMockProviders() []Provider {
	now := time.Now()
	h1 := now.Add(4 * time.Hour).UnixMilli()
	h5 := now.Add(12 * time.Hour).UnixMilli()
	d3 := now.Add(3 * time.Hour).UnixMilli()
	w5 := now.Add(5 * 24 * time.Hour).UnixMilli()

	pct := func(v float64) *float64 { return &v }
	reset := func(v int64) *int64 { return &v }

	return []Provider{
		&mockProvider{
			name:       "智谱 GLM (测试)",
			shortLabel: "智谱",
			windows: []QuotaWindow{
				{
					Label:       "时度",
					Percentage:  pct(18),   // 🟢 green 0-25%
					NextResetMs: reset(h1),
				},
				{
					Label:       "周度",
					Percentage:  pct(42),   // 🟡 yellow 25-50%
					NextResetMs: reset(w5),
				},
				{
					Label:       "月度",
					Percentage:  pct(68),   // 🟠 orange 50-75%
					NextResetMs: reset(d3),
				},
			},
		},
		&mockProvider{
			name:       "DeepSeek (测试)",
			shortLabel: "DS",
			windows: []QuotaWindow{
				{
					Label:       "余额",
					Percentage:  pct(88),   // 🔴 red 75-100%
					NextResetMs: reset(h5),
				},
			},
		},
		&mockProvider{
			name:       "未配置 (测试)",
			shortLabel: "空",
			windows: []QuotaWindow{
				{
					Label:       "时度",
					Percentage:  nil,       // ⬜ gray (no data)
					NextResetMs: nil,
				},
				{
					Label:       "周度",
					Percentage:  pct(95),   // 🔴 red edge case
					NextResetMs: reset(d3),
				},
			},
		},
	}
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
