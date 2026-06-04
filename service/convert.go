package service

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ConvertVideoToMP3
// video → audio/YM260124T0936.mp3
// numbered: true → YM년월일N001, YM년월일N002 ...
// numbered: false → 기존 YM년월일T시분
// startSec: 시작 위치 (초 단위), 0이면 전체 시작
// durationSec: 변환 길이 (초 단위), 0이면 전체 길이
func ConvertVideoToMP3(
	ffmpegPath string,
	videoPath string,
	outputDir string,
	bitrate string,
	numbered bool,
	index int, // numbered=true일 때 사용
	startSec int, // 분할 변환 시작 위치
	durationSec int, // 분할 길이
	logFilePath string,
	onProgress func(currentSec float64), // 변환 진행 위치(초) 콜백. nil이면 미사용
) (string, error) {

	outDir := outputDir
	if outDir == "" {
		outDir = filepath.Join(filepath.Dir(videoPath), "audio")
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", err
	}

	var fileName string
	now := time.Now()
	if numbered {
		// Step4 분할 시: YM년월일N001
		fileName = fmt.Sprintf(
			"YM%02d%02d%02dN%03d.mp3",
			now.Year()%100,
			now.Month(),
			now.Day(),
			index,
		)
	} else {
		// Step2 single 모드: 기존 방식
		fileName = fmt.Sprintf(
			"YM%02d%02d%02dT%02d%02d.mp3",
			now.Year()%100,
			now.Month(),
			now.Day(),
			now.Hour(),
			now.Minute(),
		)
	}

	outPath := filepath.Join(outDir, fileName)

	// FFmpeg args 구성
	// -nostats        : stderr의 \r 기반 통계 스팸 제거 (이게 누적되면 Scanner 64KB
	//                   토큰 한계를 넘겨 파이프 데드락 → 긴 영상 변환이 멈춤)
	// -progress pipe:1: 진행 상황을 stdout으로 줄바꿈 구분 key=value 형태로 출력
	//                   (stderr 로그는 기존대로 event.log로 유지)
	args := []string{"-y", "-nostats", "-progress", "pipe:1"}
	// 1️⃣ 빠른 seek (input 앞에 -ss)
	if startSec > 0 {
		args = append(args, "-ss", FormatTimestamp((startSec))) //fmt.Sprintf("%d", startSec))
	}

	args = append(args,
		"-i", videoPath,
	)

	// 2️⃣ duration은 input 뒤
	if durationSec > 0 {
		args = append(args, "-t", fmt.Sprintf("%d", durationSec))
	}

	args = append(args,
		"-vn",
		"-c:a", "libmp3lame",
		"-b:a", bitrate,
		"-map_metadata", "-1",
		"-avoid_negative_ts", "make_zero",
		outPath,
	)

	// if logfn != nil {
	// 	//logfn(fmt.Sprintf("▶ ffmpeg 실행: %v", args))
	// 	logfn("▶ ffmpeg 실행")
	// }

	cmd := exec.Command(ffmpegPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} // Windows 전용

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	if err := cmd.Start(); err != nil { // 🔥 반드시 필요
		return "", err
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// stdout: -progress 출력 → 진행 위치 파싱 (로그에는 남기지 않음)
	go func() {
		defer wg.Done()
		streamProgress(stdout, onProgress)
	}()

	// stderr: 기존 ffmpeg 로그 → event.log
	go func() {
		defer wg.Done()
		streamLogToFile(stderr, logFilePath)
	}()

	err = cmd.Wait() // 프로세스 종료 대기
	wg.Wait()        // 로그 완전 소진 대기

	if err != nil {
		return "", err
	}

	return outPath, nil
}

// =======================
// helpers
// =======================
func streamLogToFile(r io.Reader, logFilePath string) {
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("event.log 열기 실패:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 긴 줄에도 토큰 오버플로 방지
	for scanner.Scan() {
		line := scanner.Text()
		file.WriteString(line + "\n") // append
	}
}

// streamProgress
// ffmpeg `-progress pipe:1` 출력(stdout)을 읽어 out_time= 값을 초로 변환해 콜백 호출.
// onProgress가 nil이어도 파이프는 끝까지 비워(drain) 프로세스가 막히지 않게 한다.
func streamProgress(r io.Reader, onProgress func(currentSec float64)) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if onProgress == nil {
			continue // drain only
		}
		if sec, ok := parseFFmpegOutTime(scanner.Text()); ok {
			onProgress(sec)
		}
	}
}

// parseFFmpegOutTime
// "out_time=00:01:30.450000" 형태의 줄을 초(float)로 변환한다.
func parseFFmpegOutTime(line string) (float64, bool) {
	const prefix = "out_time="
	if !strings.HasPrefix(line, prefix) {
		return 0, false
	}
	v := strings.TrimSpace(line[len(prefix):])
	if v == "" || v == "N/A" {
		return 0, false
	}
	parts := strings.Split(v, ":")
	if len(parts) != 3 {
		return 0, false
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	s, err3 := strconv.ParseFloat(parts[2], 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, false
	}
	return float64(h*3600+m*60) + s, true
}
