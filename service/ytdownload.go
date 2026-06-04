package service

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// DownloadVideo yt-dlp를 이용해 영상 다운로드
// onLog: 다운로드 중 출력되는 로그를 전달
func DownloadVideo(
	ytdlpPath string,
	url string,
	outputDir string,
	logFilePath string,
	onLog func(string),
	onProgress func(float64), // 0.0 ~ 1.0
) (string, error) {

	if ytdlpPath == "" {
		return "", errors.New("yt-dlp path not set.")
	}
	if url == "" {
		return "", errors.New("No URL provided.")
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}

	outputTemplate := filepath.Join(outputDir, "video.%(ext)s")
	finalPath := filepath.Join(outputDir, "video.mp4")

	// 기존 video.mp4가 다른 프로세스(미리보기 ffplay 등)에 잡혀 있으면
	// yt-dlp의 --force-overwrites가 [WinError 32]로 실패한다.
	// Go에서 먼저 삭제를 시도하되, 방금 종료한 프로세스의 핸들 해제가 지연될 수 있어
	// 짧게 재시도한다. 끝까지 실패하면 외부 프로그램이 잡고 있는 것이므로 명확히 안내.
	if err := removeWithRetry(finalPath, 15, 200*time.Millisecond); err != nil {
		return "", fmt.Errorf(
			"기존 파일을 삭제할 수 없습니다. 다른 프로그램(미리보기/플레이어 등)이 사용 중인지 확인 후 닫아주세요: %s (%v)",
			finalPath, err,
		)
	}

	onLog("📥 Starting Download")

	args := []string{
		"--newline",
		"--no-playlist",
		"--force-overwrites",
		"-f", "bv*+ba/b",
		"--merge-output-format", "mp4",
		"-o", outputTemplate,
		url,
	}

	cmd := exec.Command(ytdlpPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	now := time.Now().Format("2006-01-02 15:04:05")
	header := fmt.Sprintf(
		"========================\n%s\n========================\n",
		now,
	)

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return "", err
	}
	defer logFile.Close()

	logFile.WriteString(header)

	writeLog := func(line string) {
		logFile.WriteString(line + "\n")
		if onLog != nil {
			if strings.HasPrefix(line, "[download]") {
				onLog(line)
			}

		}
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// stdout 읽기
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			writeLog(line)

			// 다운로드 퍼센트 추출
			if onProgress != nil {
				if percent := parseDownloadPercent(line); percent >= 0 {
					onProgress(percent)
				}
			}
		}
	}()

	// stderr 읽기
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			writeLog(scanner.Text())
		}
	}()

	wg.Wait()
	if err := cmd.Wait(); err != nil {
		return "", err
	}

	// 다운로드 완료 시 100%
	if onProgress != nil {
		onProgress(1.0)
	}

	if _, err := os.Stat(finalPath); err != nil {
		return "",
			fmt.Errorf("Download File not found: %s", finalPath)
	}

	onLog(fmt.Sprintf("✅ Download completed: %s", finalPath))

	return finalPath, nil
}

// ----------------------------
// 기존 파일 삭제 (재시도)
// ----------------------------
// Windows에서는 다른 프로세스가 연 파일을 삭제할 수 없다. 방금 Kill한
// 프로세스의 핸들 해제가 지연될 수 있으므로 짧게 재시도한다.
func removeWithRetry(path string, attempts int, delay time.Duration) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // 없으면 할 일 없음
	}

	var lastErr error
	for i := 0; i < attempts; i++ {
		err := os.Remove(path)
		if err == nil || os.IsNotExist(err) {
			return nil
		}
		lastErr = err
		time.Sleep(delay)
	}
	return lastErr
}

// ----------------------------
// 다운로드 퍼센트 추출
// ----------------------------
func parseDownloadPercent(line string) float64 {
	re := regexp.MustCompile(`(\d+(\.\d+)?)%`)
	match := re.FindStringSubmatch(line)
	if len(match) >= 2 {
		p, err := strconv.ParseFloat(match[1], 64)
		if err == nil {
			return p / 100
		}
	}
	return -1
}
