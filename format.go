package main

import (
	"fmt"
	"strings"
)

func formatCount(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

func formatDuration(seconds int64) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	m := seconds / 60
	if m < 60 {
		return fmt.Sprintf("%dm", m)
	}
	h := m / 60
	m = m % 60
	if h < 24 {
		if m > 0 {
			return fmt.Sprintf("%dh %dm", h, m)
		}
		return fmt.Sprintf("%dh", h)
	}
	d := h / 24
	h = h % 24
	if h > 0 {
		return fmt.Sprintf("%dd %dh", d, h)
	}
	return fmt.Sprintf("%dd", d)
}

const barWidth = 10

func formatBar(fraction *float64) string {
	if fraction == nil {
		return strings.Repeat("░", barWidth)
	}
	f := *fraction
	if f < 0 {
		f = 0
	}
	if f > 1 {
		f = 1
	}
	filled := int(f * float64(barWidth))
	partial := f*float64(barWidth) - float64(filled)
	blocks := strings.Repeat("█", filled)
	if partial >= 0.3 && filled < barWidth {
		blocks += "▓"
		filled++
	}
	blocks += strings.Repeat("░", barWidth-filled)
	return blocks
}

func statusDot(s ProviderStatus) string {
	switch s {
	case StatusOK:
		return "🟢"
	case StatusWarning:
		return "🟡"
	case StatusCritical:
		return "🔴"
	case StatusError:
		return "⚫"
	default:
		return "⚪"
	}
}

func maskKey(k string) string {
	if len(k) > 10 {
		return k[:8] + "…" + k[len(k)-4:]
	}
	if k == "" {
		return "未配置"
	}
	return "已配置"
}
