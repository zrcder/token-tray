package main

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"fyne.io/systray"
)

const refreshInterval = 60 * time.Second

var (
	cfg       Config
	providers []Provider

	mHeader   *systray.MenuItem
	winLabels [6]*systray.MenuItem
	winValues [6]*systray.MenuItem
	winResets [6]*systray.MenuItem
	mError    *systray.MenuItem
	mUpdated  *systray.MenuItem
	mRefresh  *systray.MenuItem

	mSetZhipu    *systray.MenuItem
	mSetDeepSeek *systray.MenuItem
	mQuit        *systray.MenuItem
)

func onReady() {
	systray.SetIcon(iconLoading)
	systray.SetTitle("")
	systray.SetTooltip("TokenTray — 加载中…")

	mHeader = systray.AddMenuItem("⚪ 加载中…", "")
	mHeader.Disable()
	systray.AddSeparator()

	for i := 0; i < 6; i++ {
		winLabels[i] = systray.AddMenuItem("", "")
		winValues[i] = systray.AddMenuItem("", "")
		winResets[i] = systray.AddMenuItem("", "")
		winLabels[i].Disable()
		winValues[i].Disable()
		winResets[i].Disable()
		winLabels[i].Hide()
		winValues[i].Hide()
		winResets[i].Hide()
	}
	systray.AddSeparator()

	mError = systray.AddMenuItem("", "")
	mError.Disable()
	mError.Hide()
	mUpdated = systray.AddMenuItem("", "")
	mUpdated.Disable()
	mUpdated.Hide()
	systray.AddSeparator()

	mRefresh = systray.AddMenuItem("↻ 立即刷新", "")
	systray.AddSeparator()
	mSetZhipu = systray.AddMenuItem("⚙ 智谱 API Key…", "")
	mSetDeepSeek = systray.AddMenuItem("⚙ DeepSeek API Key…", "")
	systray.AddSeparator()
	mQuit = systray.AddMenuItem("退出", "")

	cfg = LoadConfig()
	rebuildProviders()

	go refreshLoop()
	go clickLoop()
}

func rebuildProviders() {
	providers = providers[:0]
	if cfg.ZhipuAPIKey != "" {
		providers = append(providers, NewZhipuProvider(cfg.ZhipuAPIKey))
	}
	if cfg.DeepSeekAPIKey != "" {
		providers = append(providers, NewDeepSeekProvider(cfg.DeepSeekAPIKey))
	}
	updateSettingsLabels()
}

func updateSettingsLabels() {
	mSetZhipu.SetTitle(fmt.Sprintf("⚙ 智谱 API Key… %s", configMark(cfg.ZhipuAPIKey)))
	mSetDeepSeek.SetTitle(fmt.Sprintf("⚙ DeepSeek API Key… %s", configMark(cfg.DeepSeekAPIKey)))
}

func configMark(key string) string {
	if key != "" {
		return "●"
	}
	return "○"
}

func onExit() {}

func refreshLoop() {
	refreshOnce()
	ticker := time.NewTicker(refreshInterval)
	for range ticker.C {
		refreshOnce()
	}
}

func clickLoop() {
	for {
		select {
		case <-mRefresh.ClickedCh:
			go refreshOnce()
		case <-mSetZhipu.ClickedCh:
			handleZhipuSettings()
		case <-mSetDeepSeek.ClickedCh:
			handleDeepSeekSettings()
		case <-mQuit.ClickedCh:
			systray.Quit()
			return
		}
	}
}

var allReports []*UsageReport

func refreshOnce() {
	if len(providers) == 0 {
		systray.SetIcon(iconLoading)
		systray.SetTitle("")
		systray.SetTooltip("TokenTray — 未配置任何 Provider")
		mHeader.SetTitle("⚫ 未配置 — 点击下方「⚙ 设置」添加 API Key")
		mError.SetTitle("   ❌ 请先配置至少一个 Provider")
		mError.Show()
		return
	}

	reports := make([]*UsageReport, 0, len(providers))
	for _, p := range providers {
		r, err := p.FetchStatus()
		if err != nil {
			r = &UsageReport{
				ProviderName: p.Name(),
				ShortLabel:   p.ShortLabel(),
				Error:        err.Error(),
				LastUpdated:  time.Now(),
			}
		}
		reports = append(reports, r)
	}
	allReports = reports
	renderMultiReport(reports)
}

func renderMultiReport(reports []*UsageReport) {
	type windowEntry struct {
		provider string
		window   QuotaWindow
	}

	var allWindows []windowEntry
	for _, r := range reports {
		if r.Error != "" {
			continue
		}
		for _, w := range r.Windows {
			allWindows = append(allWindows, windowEntry{r.ProviderName, w})
		}
	}

	sort.Slice(allWindows, func(i, j int) bool {
		fi := allWindows[i].window.Fraction()
		fj := allWindows[j].window.Fraction()
		if fi == nil {
			return false
		}
		if fj == nil {
			return true
		}
		return *fi > *fj
	})

	topN := allWindows
	if len(topN) > 3 {
		topN = topN[:3]
	}

	segments := make([]DotColor, 0, len(topN))
	for _, we := range topN {
		segments = append(segments, colorForFraction(we.window.Fraction()))
	}
	if len(segments) == 0 {
		segments = []DotColor{colGray}
	}

	systray.SetIcon(generateSegmentedIcon(segments))
	systray.SetTitle("")

	var tips []string
	for _, r := range reports {
		if r.Error != "" {
			tips = append(tips, fmt.Sprintf("%s: ❌", r.ProviderName))
			continue
		}
		var parts []string
		for _, w := range r.Windows {
			pct := "—"
			if w.Percentage != nil {
				pct = fmt.Sprintf("%.0f%%", *w.Percentage)
			}
			parts = append(parts, pct)
		}
		tips = append(tips, fmt.Sprintf("%s %s", r.ProviderName, strings.Join(parts, "/")))
	}
	systray.SetTooltip(strings.Join(tips, " | "))

	for i := range winLabels {
		winLabels[i].SetTitle("")
		winLabels[i].Hide()
	}
	for i := range winValues {
		winValues[i].Hide()
	}
	for i := range winResets {
		winResets[i].Hide()
	}

	slotIdx := 0
	for ri, r := range reports {
		if ri > 0 {
			setSlot(slotIdx, "   ───────────────", true)
			slotIdx++
		}

		statusIcon := statusDot(r.Status())
		if r.Error != "" {
			setSlot(slotIdx, fmt.Sprintf("   %s %s ❌ %s", statusIcon, r.ProviderName, r.Error), true)
			slotIdx++
			continue
		}

		level := ""
		if r.PlanLevel != "" {
			level = " · " + strings.ToUpper(r.PlanLevel)
		}
		setSlot(slotIdx, fmt.Sprintf("   %s %s%s", statusIcon, r.ProviderName, level), true)
		slotIdx++

		for _, w := range r.Windows {
			if slotIdx >= 6 {
				break
			}
			pctStr := "—"
			if w.Percentage != nil {
				pctStr = fmt.Sprintf("%.0f%%", *w.Percentage)
			}
			bar := formatBar(w.Fraction())
			counts := ""
			if w.Used != nil && w.Limit != nil {
				counts = fmt.Sprintf("  (%s / %s)", formatCount(*w.Used), formatCount(*w.Limit))
			}
			setSlot(slotIdx, fmt.Sprintf("      %s %s  %s%s", w.Label, bar, pctStr, counts), true)
			slotIdx++
		}
	}

	mError.Hide()
	mUpdated.SetTitle("   更新于 " + time.Now().Format("15:04:05"))
	mUpdated.Show()
}

func setSlot(idx int, title string, show bool) {
	if idx >= 6 {
		return
	}
	winLabels[idx].SetTitle(title)
	if show {
		winLabels[idx].Show()
	} else {
		winLabels[idx].Hide()
	}
}

func handleZhipuSettings() {
	newKey := promptDialog(
		"智谱 API Key",
		fmt.Sprintf("当前: %s\n请输入智谱 API Key:", maskKey(cfg.ZhipuAPIKey)),
	)
	if newKey == "__CANCELLED__" {
		return
	}
	cfg.ZhipuAPIKey = newKey
	_ = SaveConfig(cfg)
	rebuildProviders()
	go refreshOnce()
}

func handleDeepSeekSettings() {
	newKey := promptDialog(
		"DeepSeek API Key",
		fmt.Sprintf("当前: %s\n请输入 DeepSeek API Key (sk-开头):", maskKey(cfg.DeepSeekAPIKey)),
	)
	if newKey == "__CANCELLED__" {
		return
	}
	cfg.DeepSeekAPIKey = newKey
	_ = SaveConfig(cfg)
	rebuildProviders()
	go refreshOnce()
}

func promptDialog(title, message string) string {
	script := fmt.Sprintf(`
set dialogResult to display dialog "%s" default answer "" with title "TokenTray — %s" buttons {"取消", "保存"} default button "保存"
if button returned of dialogResult = "保存" then
	return text returned of dialogResult
end if
return "__CANCELLED__"
`, escapeDialog(message), escapeDialog(title))

	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return "__CANCELLED__"
	}
	return strings.TrimSpace(strings.TrimRight(string(out), "\n"))
}

func escapeDialog(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
