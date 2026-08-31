package procx

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

func SignalKittyUSR1() error {
	pids, err := kittyPIDs()
	if err != nil {
		return err
	}
	if len(pids) == 0 {
		return fmt.Errorf("no kitty process found")
	}

	var last error
	signaled := 0
	for _, pid := range pids {
		if err := syscall.Kill(pid, syscall.SIGUSR1); err != nil {
			last = err
			continue
		}
		signaled++
	}
	if signaled == 0 {
		if last != nil {
			return last
		}
		return fmt.Errorf("failed to signal kitty")
	}
	return nil
}

func kittyPIDs() ([]int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	self := os.Getpid()
	var pids []int
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid == self {
			continue
		}
		comm, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "comm"))
		if err != nil {
			continue
		}
		if string(bytes.TrimSpace(comm)) == "kitty" {
			pids = append(pids, pid)
		}
	}
	return pids, nil
}
