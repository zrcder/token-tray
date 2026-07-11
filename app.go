package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"fyne.io/systray"
)

const refreshInterval = 60 * time.Second

var (
	cfg      Config
	provider *ZhipuProvider

	mHeader   *systray.MenuItem
	winLabels [3]*systray.MenuItem
	winValues [3]*systray.MenuItem
	winResets [3]*systray.MenuItem
	mError    *systray.MenuItem
	mUpdated  *systray.MenuItem
	mRefresh  *systray.MenuItem
	mSettings *systray.MenuItem
	mQuit     *systray.MenuItem
)

func onReady() {
	systray.SetIcon(iconLoading)
	systray.SetTitle("")
	systray.SetTooltip("TokenTray — 加载中…")

	mHeader = systray.AddMenuItem("⚪ 加载中…", "")
	mHeader.Disable()
	systray.AddSeparator()

	for i := 0; i < 3; i++ {
		winLabels[i] = systray.AddMenuItem("", "")
		winValues[i] = systray.AddMenuItem("", "")
		winResets[i] = systray.AddMenuItem("", "")
		winLabels[i].Disable()
		winValues[i].Disable()
		winResets[i].Disable()
		winLabels[i].Hide()
		winValues[i].Hide()
		winResets[i].Hide()
		systray.AddSeparator()
	}

	mError = systray.AddMenuItem("", "")
	mError.Disable()
	mError.Hide()
	mUpdated = systray.AddMenuItem("", "")
	mUpdated.Disable()
	mUpdated.Hide()
	systray.AddSeparator()

	mRefresh = systray.AddMenuItem("↻ 立即刷新", "")
	mSettings = systray.AddMenuItem("⚙ 设置…", "")
	mQuit = systray.AddMenuItem("退出", "")

	cfg = LoadConfig()
	if cfg.ZhipuAPIKey != "" {
		provider = NewZhipuProvider(cfg.ZhipuAPIKey)
	}

	go refreshLoop()
	go clickLoop()
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
		case <-mSettings.ClickedCh:
			handleSettings()
		case <-mQuit.ClickedCh:
			systray.Quit()
			return
		}
	}
}

func refreshOnce() {
	if provider == nil {
		systray.SetIcon(iconLoading)
		systray.SetTitle("")
		systray.SetTooltip("TokenTray — 未配置 API Key")
		mHeader.SetTitle("⚫ 未配置 — 点击「⚙ 设置…」")
		mError.SetTitle("   ❌ 请点击「⚙ 设置…」填入 API Key")
		mError.Show()
		return
	}

	report, err := provider.FetchStatus()
	if err != nil {
		systray.SetIcon(iconLoading)
		systray.SetTitle("")
		systray.SetTooltip("TokenTray — " + err.Error())
		mHeader.SetTitle("⚫ 智谱 GLM")
		mError.SetTitle("   ❌ " + err.Error())
		mError.Show()
		mUpdated.SetTitle("   更新于 " + time.Now().Format("15:04:05"))
		mUpdated.Show()
		return
	}
	renderReport(report)
}

func renderReport(r *UsageReport) {
	segments := make([]DotColor, 0, len(r.Windows))
	for _, w := range r.Windows {
		segments = append(segments, colorForFraction(w.Fraction()))
	}
	if len(segments) == 0 {
		segments = []DotColor{colGray}
	}
	systray.SetIcon(generateSegmentedIcon(segments))
	systray.SetTitle("")

	var parts []string
	for _, w := range r.Windows {
		pct := "—"
		if w.Percentage != nil {
			pct = fmt.Sprintf("%.0f%%", *w.Percentage)
		}
		parts = append(parts, fmt.Sprintf("%s %s", w.Label, pct))
	}
	systray.SetTooltip(strings.Join(parts, " | "))

	level := ""
	if r.PlanLevel != "" {
		level = " · " + strings.ToUpper(r.PlanLevel)
	}
	mHeader.SetTitle(fmt.Sprintf("%s %s%s", statusDot(r.Status()), r.ProviderName, level))

	// Windows
	for i := 0; i < 3; i++ {
		if i < len(r.Windows) {
			w := r.Windows[i]
			winLabels[i].SetTitle("   " + w.Label)
			winLabels[i].Show()

			pctStr := "—"
			if w.Percentage != nil {
				pctStr = fmt.Sprintf("%.0f%%", *w.Percentage)
			}
			bar := formatBar(w.Fraction())
			counts := ""
			if w.Used != nil && w.Limit != nil {
				counts = fmt.Sprintf("  (%s / %s)", formatCount(*w.Used), formatCount(*w.Limit))
			}
			winValues[i].SetTitle(fmt.Sprintf("   %s  %s%s", bar, pctStr, counts))
			winValues[i].Show()

			if reset := w.ResetInSeconds(); reset != nil {
				winResets[i].SetTitle(fmt.Sprintf("   ⏳ %s 后重置", formatDuration(*reset)))
				winResets[i].Show()
			} else {
				winResets[i].Hide()
			}
		} else {
			winLabels[i].Hide()
			winValues[i].Hide()
			winResets[i].Hide()
		}
	}

	mError.Hide()
	mUpdated.SetTitle("   更新于 " + r.LastUpdated.Format("15:04:05"))
	mUpdated.Show()
}

func handleSettings() {
	currentHint := maskKey(cfg.ZhipuAPIKey)
	script := fmt.Sprintf(`
set dialogResult to display dialog "当前: %s\n请输入智谱 API Key:" default answer "" with title "TokenTray 设置" buttons {"取消", "保存"} default button "保存"
if button returned of dialogResult = "保存" then
	return text returned of dialogResult
end if
return "__CANCELLED__"
`, currentHint)

	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return
	}
	newKey := strings.TrimSpace(strings.TrimRight(string(out), "\n"))
	if newKey == "__CANCELLED__" {
		return
	}

	if newKey == "" {
		confirmScript := `display dialog "确定要清空已配置的 API Key 吗？" with title "确认" buttons {"取消", "清空"} default button "取消"`
		confirmOut, err := exec.Command("osascript", "-e", confirmScript).Output()
		if err != nil || strings.TrimSpace(string(confirmOut)) != "清空" {
			return
		}
	}

	cfg.ZhipuAPIKey = newKey
	if err := SaveConfig(cfg); err != nil {
		return
	}
	if newKey == "" {
		provider = nil
	} else {
		provider = NewZhipuProvider(newKey)
	}
	go refreshOnce()
}
