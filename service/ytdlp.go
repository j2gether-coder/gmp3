package service

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"syscall"
)

const ytDlpURL = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe"

func EnsureYtDlp(binDir string) (string, error) {
	ytdlpPath := filepath.Join(binDir, "yt-dlp.exe")

	if FileExists(ytdlpPath) {
		return ytdlpPath, nil
	}

	if err := ChunkDownload(ytDlpURL, ytdlpPath); err != nil {
		return "", err
	}

	// sanity check
	cmd := exec.Command(ytdlpPath, "--version")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} // Windows 전용
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("yt-dlp 실행 실패")
	}

	return ytdlpPath, nil
}
