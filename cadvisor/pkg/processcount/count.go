// Package processcount counts configured process names from the host procfs.
package processcount

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Collect uses exact /proc/PID/status Name matches, just like kube_process.
// Linux supplies the truncated comm name; configured names are not rewritten.
// An incomplete scan returns an error, never fabricated zero counts.
func Collect(procPath string, targets []string) (map[string]int, error) {
	counts := make(map[string]int, len(targets))
	for _, target := range targets {
		if target != "" {
			counts[target] = 0
		}
	}
	if len(counts) == 0 {
		return counts, nil
	}
	entries, err := os.ReadDir(procPath)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 || pid == 2 {
			continue
		}
		data, err := os.ReadFile(filepath.Join(procPath, entry.Name(), "status"))
		if os.IsNotExist(err) {
			continue
		} // Process exited during the scan.
		if err != nil {
			return nil, err
		}
		name, ppid := "", -1
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "Name:") {
				name = strings.TrimPrefix(line, "Name:\t")
			}
			if strings.HasPrefix(line, "PPid:") {
				ppid, err = strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "PPid:")))
				if err != nil {
					return nil, fmt.Errorf("invalid PPid for PID %d", pid)
				}
			}
		}
		if name == "" || ppid < 0 {
			return nil, fmt.Errorf("incomplete status for PID %d", pid)
		}
		if ppid == 2 {
			continue
		} // Same kernel-thread exclusion as kube_process.
		if _, ok := counts[name]; ok {
			counts[name]++
		}
	}
	return counts, nil
}
