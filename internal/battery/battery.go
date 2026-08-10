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
)

type Status struct {
	Health      string  `json:"health"`
	Percentage  int     `json:"percentage"`
	Plugged     string  `json:"plugged"`
	Status      string  `json:"status"`
	Temperature float64 `json:"temperature"`
	Current     int     `json:"current"`
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
