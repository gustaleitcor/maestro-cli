// Package humanize writes sizes, rates and durations the way top does.
package humanize

import (
	"fmt"
	"time"
)

// Ago says how long before now t was, as a list shows it: "just now",
// "5m ago", "3h ago", "4d ago", and the day itself once it is a week old.
func Ago(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
	return t.Local().Format("2006-01-02")
}

// In says how long after now t is, as a list shows it: "in 3d", "in 5h",
// and "within the hour" for less than an hour, or a time already past.
func In(t, now time.Time) string {
	d := t.Sub(now)
	switch {
	case d < time.Hour:
		return "within the hour"
	case d < 24*time.Hour:
		return fmt.Sprintf("in %dh", int(d.Hours()))
	}
	return fmt.Sprintf("in %dd", int(d.Hours()/24))
}

// Bytes writes n in binary units, to three digits: 512B, 1.5K, 16G.
func Bytes(n uint64) string {
	const units = "KMGTPE"
	if n < 1024 {
		return fmt.Sprintf("%dB", n)
	}
	value, unit := float64(n), -1
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if value < 10 {
		return fmt.Sprintf("%.1f%c", value, units[unit])
	}
	return fmt.Sprintf("%.0f%c", value, units[unit])
}

// Megabytes writes mb megabytes as Bytes does: 512M, 16G.
func Megabytes(mb int64) string {
	return Bytes(uint64(max(mb, 0)) << 20)
}

func Rate(bytesPerSecond float64) string {
	if bytesPerSecond < 0 {
		bytesPerSecond = 0
	}
	return Bytes(uint64(bytesPerSecond)) + "/s"
}

func Uptime(seconds float64) string {
	d := time.Duration(seconds) * time.Second
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
	}
	return fmt.Sprintf("%dd%dh", int(d.Hours())/24, int(d.Hours())%24)
}

// Limit is seconds as maestro.toml's time writes them: 2h, 90m, 45s.
func Limit(seconds int64) string {
	switch {
	case seconds%3600 == 0:
		return fmt.Sprintf("%dh", seconds/3600)
	case seconds%60 == 0:
		return fmt.Sprintf("%dm", seconds/60)
	}
	return fmt.Sprintf("%ds", seconds)
}

func Percent(p float64) string {
	return fmt.Sprintf("%.0f%%", p)
}

// Fraction is used over total, as a percentage; zero when there is no total.
func Fraction(used, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return 100 * float64(used) / float64(total)
}
