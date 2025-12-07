package checker

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"gocheck/internal/models"
)

func (e *Engine) performPingCheck(check *models.Check, history *models.CheckHistory, start time.Time) {
	host := check.Host
	if host == "" {
		history.Success = false
		history.ErrorMessage = "no host specified"
		history.ResponseTimeMs = int(time.Since(start).Milliseconds())
		return
	}

	timeout := time.Duration(check.TimeoutSeconds) * time.Second

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("ping", "-n", "1", "-w", fmt.Sprintf("%d", check.TimeoutSeconds*1000), host)
	} else {
		cmd = exec.Command("ping", "-c", "1", "-W", fmt.Sprintf("%d", check.TimeoutSeconds), host)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)

	output, err := cmd.CombinedOutput()
	history.ResponseTimeMs = int(time.Since(start).Milliseconds())

	if err != nil {
		history.Success = false
		history.ErrorMessage = fmt.Sprintf("ping failed: %v", err)
		return
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "time=") || strings.Contains(outputStr, "Time=") {
		re := regexp.MustCompile(`time[=<](\d+\.?\d*)`)
		matches := re.FindStringSubmatch(outputStr)
		if len(matches) > 1 {
			history.Success = true
		} else {
			history.Success = true
		}
	} else if strings.Contains(outputStr, "bytes from") || strings.Contains(outputStr, "Reply from") {
		history.Success = true
	} else {
		history.Success = false
		history.ErrorMessage = "no response from host"
	}
}
