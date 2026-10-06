// Package humanize writes sizes, rates and durations the way top does.
package humanize

import (
	"fmt"
	"time"
)

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

// Rate writes bytes per second.
func Rate(bytesPerSecond float64) string {
	if bytesPerSecond < 0 {
		bytesPerSecond = 0
	}
	return Bytes(uint64(bytesPerSecond)) + "/s"
}

// Uptime writes how long something has been up: 12m, 5h12m, 3d4h.
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

// Percent writes a percentage without decimals.
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
