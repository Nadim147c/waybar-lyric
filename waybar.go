package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

const SIGRTMIN = 34

func UpdateWaybar() error {
	return SendSignal("^waybar$", SIGRTMIN+4)
}

func SendSignal(processName string, signal int) error {
	cmd := exec.Command("pgrep", processName)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to find processes matching %q: %w", processName, err)
	}

	pidStrings := strings.Fields(string(output))
	if len(pidStrings) == 0 {
		return fmt.Errorf("no processes found matching %q", processName)
	}

	for _, pidStr := range pidStrings {
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			return fmt.Errorf("invalid PID %q: %w", pidStr, err)
		}

		// Send the signal
		err = syscall.Kill(pid, syscall.Signal(signal))
		if err != nil {
			return fmt.Errorf("failed to send signal to PID %d: %w", pid, err)
		}
	}

	return nil
}
