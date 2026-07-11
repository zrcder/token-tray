package main

import "time"

type QuotaWindow struct {
	Label       string
	Used        *int64   // currentValue (count) — may be nil for %-only responses.
	Limit       *int64   // usage (total).
	Percentage  *float64 // 0-100, preferred when available.
	NextResetMs *int64   // epoch milliseconds.
}

func (w QuotaWindow) Fraction() *float64 {
	if w.Percentage != nil {
		f := *w.Percentage / 100.0
		return &f
	}
	if w.Used != nil && w.Limit != nil && *w.Limit > 0 {
		f := float64(*w.Used) / float64(*w.Limit)
		return &f
	}
	return nil
}

func (w QuotaWindow) ResetInSeconds() *int64 {
	if w.NextResetMs == nil {
		return nil
	}
	now := time.Now().UnixMilli()
	diff := *w.NextResetMs - now
	if diff < 0 {
		diff = 0
	}
	secs := diff / 1000
	return &secs
}

type UsageReport struct {
	ProviderName string
	ShortLabel   string
	PlanLevel    string
	Windows      []QuotaWindow
	Error        string
	LastUpdated  time.Time
}

type ProviderStatus int

const (
	StatusUnknown ProviderStatus = iota
	StatusOK
	StatusWarning
	StatusCritical
	StatusError
)

func (r *UsageReport) Status() ProviderStatus {
	if r.Error != "" {
		return StatusError
	}
	var maxFrac float64 = -1
	for _, w := range r.Windows {
		if f := w.Fraction(); f != nil && *f > maxFrac {
			maxFrac = *f
		}
	}
	if maxFrac < 0 {
		return StatusUnknown
	}
	switch {
	case maxFrac >= 0.9:
		return StatusCritical
	case maxFrac >= 0.7:
		return StatusWarning
	default:
		return StatusOK
	}
}

type Provider interface {
	Name() string
	ShortLabel() string
	FetchStatus() (*UsageReport, error)
}
