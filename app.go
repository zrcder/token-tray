package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"fyne.io/systray"
)

const appVersion = "v0.1.0"

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

	mZhipuLabel   *systray.MenuItem
	mZhipuEdit    *systray.MenuItem
	mZhipuDelete  *systray.MenuItem
	mDSLabel      *systray.MenuItem
	mDSEdit       *systray.MenuItem
	mDSDelete     *systray.MenuItem
	mInterval     *systray.MenuItem
	mAbout        *systray.MenuItem
	mQuit         *systray.MenuItem

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
	mInterval = systray.AddMenuItem(intervalLabel(), "")
	systray.AddSeparator()

	mZhipuLabel = systray.AddMenuItem("", "")
	mZhipuLabel.Disable()
	mZhipuEdit = systray.AddMenuItem("", "")
	mZhipuDelete = systray.AddMenuItem("", "")
	mDSLabel = systray.AddMenuItem("", "")
	mDSLabel.Disable()
	mDSEdit = systray.AddMenuItem("", "")
	mDSDelete = systray.AddMenuItem("", "")
	systray.AddSeparator()

	mAbout = systray.AddMenuItem(fmt.Sprintf("关于 TokenTray %s", appVersion), "")
	mQuit = systray.AddMenuItem("退出", "")

	mu.Lock()
	if testMode {
		providers = makeMockProviders()
		hideSettingsMenu()
	} else {
		cfg = LoadConfig()
		rebuildProvidersLocked()
	}
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
	updateProviderMenu(mZhipuLabel, mZhipuEdit, mZhipuDelete, "智谱 GLM", cfg.ZhipuAPIKey)
	updateProviderMenu(mDSLabel, mDSEdit, mDSDelete, "DeepSeek", cfg.DeepSeekAPIKey)
}

func updateProviderMenu(label, edit, delete_ *systray.MenuItem, name, key string) {
	if key != "" {
		label.SetTitle(fmt.Sprintf("  %s  ● 已配置", name))
		edit.SetTitle("    ✎ 修改 API Key…")
		delete_.SetTitle("    ✕ 删除")
		delete_.Show()
	} else {
		label.SetTitle(fmt.Sprintf("  %s  ○ 未配置", name))
		edit.SetTitle("    ✎ 添加 API Key…")
		delete_.Hide()
	}
}

func intervalLabel() string {
	m := intervalOptions[intervalIdx].Minutes()
	if m < 1 {
		return fmt.Sprintf("⏱ 刷新间隔: %d秒", int(intervalOptions[intervalIdx].Seconds()))
	}
	return fmt.Sprintf("⏱ 刷新间隔: %d分钟", int(m))
}

func onExit() {}

func hideSettingsMenu() {
	mZhipuLabel.Hide()
	mZhipuEdit.Hide()
	mZhipuDelete.Hide()
	mDSLabel.Hide()
	mDSEdit.Hide()
	mDSDelete.Hide()
}

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
		mError.SetTitle("⚠ 请点击下方「✎ 添加 API Key」配置 Provider")
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
		case <-mZhipuEdit.ClickedCh:
			if !testMode {
				go handleEditProvider("智谱 GLM", &cfg.ZhipuAPIKey, true)
			}
		case <-mZhipuDelete.ClickedCh:
			if !testMode {
				go handleDeleteProvider("智谱 GLM", &cfg.ZhipuAPIKey)
			}
		case <-mDSEdit.ClickedCh:
			if !testMode {
				go handleEditProvider("DeepSeek", &cfg.DeepSeekAPIKey, false)
			}
		case <-mDSDelete.ClickedCh:
			if !testMode {
				go handleDeleteProvider("DeepSeek", &cfg.DeepSeekAPIKey)
			}
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
		case <-mAbout.ClickedCh:
			go handleAbout()
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
			setSlot(slotIdx, fmt.Sprintf("   %s  %s  %s%s", w.Label, bar, pctStr, reset), true)
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

func handleEditProvider(name string, keyPtr *string, isZhipu bool) {
	hint := "请输入 API Key:"
	if isZhipu {
		hint = "请输入智谱 API Key:"
	} else {
		hint = "请输入 DeepSeek API Key (sk-开头):"
	}
	prompt := fmt.Sprintf("当前: %s\n%s", maskKey(*keyPtr), hint)

	newKey := promptDialog(name, prompt)
	if newKey == "__CANCELLED__" {
		return
	}
	mu.Lock()
	*keyPtr = newKey
	if err := SaveConfig(cfg); err != nil {
		mu.Unlock()
		mError.SetTitle(fmt.Sprintf("❌ 保存失败: %v", err))
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

func handleDeleteProvider(name string, keyPtr *string) {
	if !confirmDialog(name, fmt.Sprintf("确定删除 %s 的 API Key？\n删除后将停止监控该 Provider。", name)) {
		return
	}
	mu.Lock()
	*keyPtr = ""
	if err := SaveConfig(cfg); err != nil {
		mu.Unlock()
		mError.SetTitle(fmt.Sprintf("❌ 保存失败: %v", err))
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

func handleAbout() {
	extra := ""
	if testMode {
		extra = "\n\n⚡ 测试模式 — 使用虚拟数据，无真实 API 调用"
	}
	infoDialog("关于",
		fmt.Sprintf("TokenTray %s\n\nmacOS / Windows / Linux 菜单栏大模型用量监控\n\n支持: 智谱 GLM · DeepSeek\n开源: github.com/zrcder/token-tray%s", appVersion, extra))
}
