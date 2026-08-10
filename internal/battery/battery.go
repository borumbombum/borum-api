// Package battery reads Android battery status via the Termux:API bridge.
//
// Requires, on the device:
//   - the `termux-api` package: pkg install termux-api
//   - the companion "Termux:API" Android app installed from the SAME
//     source as Termux itself (F-Droid or GitHub — not Play Store),
//     see https://github.com/termux/termux-api
//
// Without both installed, Get() will hang or fail since the CLI tool
// blocks waiting for a response from the Termux:API app
package battery

import (
	"encoding/json"
	"os/exec"
	"strings"
	"sync"
)

type Status struct {
	Health      string  `json:"health"`
	Percentage  int     `json:"percentage"`
	Plugged     string  `json:"plugged"`
	Status      string  `json:"status"`
	Temperature float64 `json:"temperature"`
	Current     int     `json:"current"`
}

// Snapshot is a cached battery reading. Available is false when the battery
// could not be read (e.g. not running on Android), so callers can render a
// placeholder instead of a stale or zero value.
type Snapshot struct {
	Status
	Available bool
}

func Get() (Status, error) {
	var s Status

	out, err := exec.Command("termux-battery-status").Output()
	if err != nil {
		return s, err
	}

	if err := json.Unmarshal(out, &s); err != nil {
		return s, err
	}

	return s, nil
}

// Refresh reads the battery and stores it as the current snapshot. On failure
// it clears the snapshot so Current reports unavailable. Safe to call from the
// task scheduler; it is the only writer to the snapshot.
func Refresh() {
	s, err := Get()

	mu.Lock()
	defer mu.Unlock()
	if err != nil {
		snapshot = Snapshot{}
		return
	}
	snapshot = Snapshot{Status: s, Available: true}
}

// Current returns the latest snapshot without blocking on the Termux app.
func Current() Snapshot {
	mu.RLock()
	defer mu.RUnlock()
	return snapshot
}

// Charging reports whether the device is plugged in and actively charging.
func (s Snapshot) Charging() bool {
	if !s.Available {
		return false
	}
	switch strings.ToLower(s.Status.Status) {
	case "charging":
		return true
	case "full", "discharging", "not_charging", "not-charging", "unknown":
		return false
	}
	return s.Plugged != ""
}

var (
	mu       sync.RWMutex
	snapshot Snapshot
)
