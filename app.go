package main

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"fyne.io/systray"
)

var (
	mu              sync.Mutex
	cfg             Config
	providers       []Provider
	refreshInterval = 5 * time.Minute
	refreshCh       = make(chan struct{}, 1)

	mHeader   *systray.MenuItem
	winLabels [12]*systray.MenuItem
	mError    *systray.MenuItem
	mUpdated  *systray.MenuItem
	mRefresh  *systray.MenuItem

	mSetZhipu    *systray.MenuItem
	mSetDeepSeek *systray.MenuItem
	mInterval    *systray.MenuItem
	mQuit        *systray.MenuItem

	intervalOptions = []time.Duration{1 * time.Minute, 5 * time.Minute, 10 * time.Minute, 30 * time.Minute}
	intervalIdx     = 1
)

func onReady() {
	systray.SetIcon(iconLoading)
	systray.SetTitle("")
	systray.SetTooltip("TokenTray — 加载中…")

	mHeader = systray.AddMenuItem("", "")
	mHeader.Disable()
	mHeader.Hide()
	systray.AddSeparator()

	for i := 0; i < 12; i++ {
		winLabels[i] = systray.AddMenuItem("", "")
		winLabels[i].Disable()
		winLabels[i].Hide()
	}
	systray.AddSeparator()

	mError = systray.AddMenuItem("", "")
	mError.Disable()
	mError.Hide()
	mUpdated = systray.AddMenuItem("", "")
	mUpdated.Disable()
	mUpdated.Hide()
	systray.AddSeparator()

	mRefresh = systray.AddMenuItem("↻ 刷新", "")
	systray.AddSeparator()
	mSetZhipu = systray.AddMenuItem("⚙ 智谱 API Key…", "")
	mSetDeepSeek = systray.AddMenuItem("⚙ DeepSeek API Key…", "")
	mInterval = systray.AddMenuItem(intervalLabel(), "")
	systray.AddSeparator()
	mQuit = systray.AddMenuItem("退出", "")

	mu.Lock()
	cfg = LoadConfig()
	rebuildProvidersLocked()
	mu.Unlock()

	go refreshLoop()
	go clickLoop()
}

func rebuildProvidersLocked() {
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

func intervalLabel() string {
	m := intervalOptions[intervalIdx].Minutes()
	if m < 1 {
		return fmt.Sprintf("⏱ 刷新间隔: %d秒", int(intervalOptions[intervalIdx].Seconds()))
	}
	return fmt.Sprintf("⏱ 刷新间隔: %d分钟", int(m))
}

func onExit() {}

func refreshLoop() {
	doRefresh()
	for {
		mu.Lock()
		interval := refreshInterval
		mu.Unlock()
		select {
		case <-time.After(interval):
			doRefresh()
		case <-refreshCh:
			doRefresh()
		}
	}
}

func doRefresh() {
	mu.Lock()
	snapshot := make([]Provider, len(providers))
	copy(snapshot, providers)
	mu.Unlock()

	if len(snapshot) == 0 {
		systray.SetIcon(iconLoading)
		systray.SetTitle("")
		systray.SetTooltip("TokenTray — 未配置")
		for i := range winLabels {
			winLabels[i].Hide()
		}
		mError.SetTitle("⚠ 请点击「⚙ 设置」添加 API Key")
		mError.Show()
		mUpdated.Hide()
		return
	}

	reports := make([]*UsageReport, 0, len(snapshot))
	for _, p := range snapshot {
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
	renderMultiReport(reports)
}

func clickLoop() {
	for {
		select {
		case <-mRefresh.ClickedCh:
			select {
			case refreshCh <- struct{}{}:
			default:
			}
		case <-mSetZhipu.ClickedCh:
			go handleZhipuSettings()
		case <-mSetDeepSeek.ClickedCh:
			go handleDeepSeekSettings()
		case <-mInterval.ClickedCh:
			mu.Lock()
			intervalIdx = (intervalIdx + 1) % len(intervalOptions)
			refreshInterval = intervalOptions[intervalIdx]
			label := intervalLabel()
			mu.Unlock()
			mInterval.SetTitle(label)
			select {
			case refreshCh <- struct{}{}:
			default:
			}
		case <-mQuit.ClickedCh:
			systray.Quit()
			return
		}
	}
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
	mHeader.Hide()

	slotIdx := 0
	for ri, r := range reports {
		if ri > 0 {
			setSlot(slotIdx, "", true)
			slotIdx++
		}

		statusIcon := statusDot(r.Status())
		if r.Error != "" {
			setSlot(slotIdx, fmt.Sprintf("%s %s — %s", statusIcon, r.ProviderName, r.Error), true)
			slotIdx++
			continue
		}

		level := ""
		if r.PlanLevel != "" {
			level = " · " + r.PlanLevel
		}
		setSlot(slotIdx, fmt.Sprintf("%s %s%s", statusIcon, r.ProviderName, level), true)
		slotIdx++

		for _, w := range r.Windows {
			if slotIdx >= 12 {
				break
			}
			pctStr := "—"
			if w.Percentage != nil {
				pctStr = fmt.Sprintf("%2.0f%%", *w.Percentage)
			}
			bar := formatBar(w.Fraction())
			reset := ""
			if s := w.ResetInSeconds(); s != nil && *s > 0 {
				reset = fmt.Sprintf("  %s", formatDuration(*s))
			}
			label := padRight(w.Label, 6)
			setSlot(slotIdx, fmt.Sprintf("   %s %s  %s%s", label, bar, pctStr, reset), true)
			slotIdx++
		}
	}

	mError.Hide()
	mUpdated.SetTitle(fmt.Sprintf("⏱ %s", time.Now().Format("15:04")))
	mUpdated.Show()
}

func setSlot(idx int, title string, show bool) {
	if idx >= 12 {
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
	mu.Lock()
	cfg.ZhipuAPIKey = newKey
	if err := SaveConfig(cfg); err != nil {
		mu.Unlock()
		mError.SetTitle(fmt.Sprintf("   ❌ 保存失败: %v", err))
		mError.Show()
		return
	}
	rebuildProvidersLocked()
	mu.Unlock()
	select {
	case refreshCh <- struct{}{}:
	default:
	}
}

func handleDeepSeekSettings() {
	newKey := promptDialog(
		"DeepSeek API Key",
		fmt.Sprintf("当前: %s\n请输入 DeepSeek API Key (sk-开头):", maskKey(cfg.DeepSeekAPIKey)),
	)
	if newKey == "__CANCELLED__" {
		return
	}
	mu.Lock()
	cfg.DeepSeekAPIKey = newKey
	if err := SaveConfig(cfg); err != nil {
		mu.Unlock()
		mError.SetTitle(fmt.Sprintf("   ❌ 保存失败: %v", err))
		mError.Show()
		return
	}
	rebuildProvidersLocked()
	mu.Unlock()
	select {
	case refreshCh <- struct{}{}:
	default:
	}
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
