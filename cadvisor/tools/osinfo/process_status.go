package osinfo

import (
	"bytes"
	"os"
	"path/filepath"
)

// readProcessStatus returns nil, nil only for a definite non-target comm.
// comm and status Name share Linux's 15-byte task name, but status escapes
// backslashes/newlines. Uncertain names and all comm errors use the old status
// path. The caller must still validate the real status Name (rename races).
// Nothing is cached: targets, names and status are reconsidered on every scan.
func readProcessStatus(pidDir string, targets []string) ([]byte, error) {
	comm, err := os.Open(filepath.Join(pidDir, "comm"))
	if err != nil {
		return os.ReadFile(filepath.Join(pidDir, "status"))
	}
	// One extra byte detects longer/unexpected names without allocating a full
	// ReadFile buffer or reading a whole status for every unrelated PID.
	var buf [17]byte
	n, readErr := comm.Read(buf[:])
	comm.Close()
	if readErr == nil && n > 1 && n <= 16 && buf[n-1] == '\n' &&
		bytes.IndexAny(buf[:n-1], "\\\n	\x00") < 0 {
		name := buf[:n-1]
		matched := false
		for _, target := range targets {
			if bytes.Equal(name, []byte(target)) {
				matched = true
				break
			}
		}
		if !matched {
			return nil, nil
		}
	}
	return os.ReadFile(filepath.Join(pidDir, "status"))
}
