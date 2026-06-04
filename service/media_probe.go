package service

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

// GetMediaDuration
// ffprobe를 이용해 미디어 전체 길이(초 단위)를 반환
func GetMediaDuration(
	ffprobePath string,
	mediaPath string,
) (float64, error) {

	cmd := exec.Command(
		ffprobePath,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		mediaPath,
	)

	// ✅ Windows CMD 창 숨김
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	var duration float64
	_, err = fmt.Sscanf(strings.TrimSpace(string(out)), "%f", &duration)
	if err != nil {
		return 0, err
	}

	return duration, nil
}
