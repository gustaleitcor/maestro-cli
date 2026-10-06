package humanize

import "testing"

func TestBytes(t *testing.T) {
	for n, want := range map[uint64]string{
		0: "0B", 512: "512B", 1024: "1.0K", 1536: "1.5K", 10 * 1024: "10K", 1 << 20: "1.0M",
		16 << 30: "16G", 40960 << 20: "40G", 3<<40 + 1<<39: "3.5T",
	} {
		if got := Bytes(n); got != want {
			t.Errorf("Bytes(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestUptime(t *testing.T) {
	for seconds, want := range map[float64]string{30: "0m", 720: "12m", 5*3600 + 12*60: "5h12m", 3*86400 + 4*3600 + 59: "3d4h"} {
		if got := Uptime(seconds); got != want {
			t.Errorf("Uptime(%v) = %q, want %q", seconds, got, want)
		}
	}
}

func TestRateAndFraction(t *testing.T) {
	if got := Rate(1536); got != "1.5K/s" {
		t.Errorf("Rate = %q", got)
	}
	if got := Rate(-3); got != "0B/s" {
		t.Errorf("Rate of a negative = %q", got)
	}
	if Fraction(1, 0) != 0 || Fraction(1, 4) != 25 {
		t.Error("Fraction")
	}
}
