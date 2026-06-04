package service

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// PlayAudio
// ffplay 기반 오디오 재생 (duration은 UI에서 Progress 용도로 사용)
func PlayAudio(
	ffplayPath string,
	logFilePath string,
	audioPath string,
	onLog func(string),
	onProgress func(float64),
	onState func(string),
	//logFailPath string,
) (*exec.Cmd, error) {
	// 실행 파일 존재 확인
	if _, err := os.Stat(ffplayPath); err != nil {
		return nil, err
	}
	if _, err := os.Stat(audioPath); err != nil {
		return nil, err
	}

	args := []string{
		"-autoexit",
		"-nodisp",
		audioPath,
	}

	cmd := exec.Command(ffplayPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true, // Windows CMD 숨김
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	// 로그 파일 한 번만 열기
	logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return nil, err
	}

	if onState != nil {
		onState("▶ Playing...")
	}

	// 로그 기록 함수
	writeLog := func(line string) {
		// if onLog != nil {
		// 	onLog(line)
		// }
		_, _ = logFile.WriteString(line + "\n")
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// stdout 읽기
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			writeLog(scanner.Text())
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

	// 진행바 애니메이션 (duration 정보 없을 때)
	go func() {
		if onProgress == nil {
			return
		}
		for i := 0.0; i < 1.0; i += 0.01 {
			onProgress(i)
			time.Sleep(100 * time.Millisecond)
		}
	}()

	// 종료 처리
	go func() {
		wg.Wait()       // stdout/stderr 다 읽고
		cmd.Wait()      // 프로세스 종료 대기
		logFile.Close() // 로그 파일 닫기

		if onState != nil {
			onState("■ Stop")
		}
		if onProgress != nil {
			onProgress(1)
		}
	}()

	return cmd, nil
}

func streamOutput(r io.Reader, logFilePath string) {

	f, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		_, _ = f.WriteString(line + "\n")
	}
}
