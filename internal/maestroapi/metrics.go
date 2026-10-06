package maestroapi

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// MachineMetrics is how loaded one machine is. Kind is "machine", or "host"
// for the server maestro-orq runs on.
type MachineMetrics struct {
	Name        string    `json:"name"`
	Kind        string    `json:"kind"`
	Status      string    `json:"status"` // ready or unreachable
	Error       string    `json:"error"`
	CollectedAt time.Time `json:"collected_at"`

	// System is empty when the machine let maestro-orq reach Podman but not
	// run commands; SystemError then says why.
	System      *SystemMetrics     `json:"system"`
	SystemError string             `json:"system_error"`
	Containers  []ContainerMetrics `json:"containers"`
}

type SystemMetrics struct {
	Hostname      string  `json:"hostname"`
	Kernel        string  `json:"kernel"`
	CPUModel      string  `json:"cpu_model"`
	UptimeSeconds float64 `json:"uptime_seconds"`

	CPU struct {
		Cores   int        `json:"cores"`
		Percent float64    `json:"percent"`
		PerCore []float64  `json:"per_core"`
		Load    [3]float64 `json:"load"`
	} `json:"cpu"`
	Memory struct {
		Total     uint64 `json:"total"`
		Used      uint64 `json:"used"`
		Available uint64 `json:"available"`
		Cached    uint64 `json:"cached"`
		SwapTotal uint64 `json:"swap_total"`
		SwapUsed  uint64 `json:"swap_used"`
	} `json:"memory"`
	Disks []struct {
		Device string `json:"device"`
		Mount  string `json:"mount"`
		Total  uint64 `json:"total"`
		Used   uint64 `json:"used"`
	} `json:"disks"`
	Network []struct {
		Name   string  `json:"name"`
		RxRate float64 `json:"rx_rate"`
		TxRate float64 `json:"tx_rate"`
	} `json:"network"`
	GPUs []struct {
		Index       int     `json:"index"`
		Name        string  `json:"name"`
		Percent     float64 `json:"percent"`
		MemoryUsed  uint64  `json:"memory_used"`
		MemoryTotal uint64  `json:"memory_total"`
		TempC       float64 `json:"temperature_c"`
		PowerW      float64 `json:"power_w"`
		PowerLimitW float64 `json:"power_limit_w"`
	} `json:"gpus"`
	Temps []struct {
		Name string  `json:"name"`
		C    float64 `json:"c"`
	} `json:"temperatures"`
}

type ContainerMetrics struct {
	Name     string  `json:"name"`
	UserID   int64   `json:"user_id"`
	Run      int64   `json:"run"`
	Line     int64   `json:"line"`
	Percent  float64 `json:"cpu_percent"` // 100 is one whole core
	MemUsed  uint64  `json:"memory_used"`
	MemLimit uint64  `json:"memory_limit"`
}

func GetMetrics(ctx context.Context, maestroKey, machine string) ([]MachineMetrics, error) {
	path := "/api/metrics"
	if machine != "" {
		path += "?machine=" + url.QueryEscape(machine)
	}
	var machines []MachineMetrics
	if err := doWithKey(ctx, maestroKey, http.MethodGet, path, nil, http.StatusOK, &machines); err != nil {
		return nil, err
	}
	return machines, nil
}
