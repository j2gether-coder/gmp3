package service

import (
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

// PlayConfig 재생 옵션
type PlayConfig struct {
	Width       int  `json:"width"`
	Height      int  `json:"height"`
	KeepAspect  bool `json:"keep_aspect"`
	WindowX     int  `json:"window_x"`
	WindowY     int  `json:"window_y"`
	AlwaysOnTop bool `json:"always_on_top"`
	Borderless  bool `json:"borderless"`
	Fullscreen  bool `json:"fullscreen"`
}

// LoadPlayConfig conf/play_video.json 로드
func LoadPlayConfig(path string) (*PlayConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg PlayConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// BuildFFplayCmd ffplay 실행 명령 생성
func BuildFFplayCmd(
	ffplayPath string,
	videoPath string,
	cfg *PlayConfig,
) (*exec.Cmd, error) {

	if cfg == nil { // nil이면 기본값으로 초기화
		cfg = &PlayConfig{}
	}

	if _, err := os.Stat(ffplayPath); err != nil {
		return nil, err
	}
	if _, err := os.Stat(videoPath); err != nil {
		return nil, err
	}

	args := []string{
		"-autoexit",
	}

	if cfg.Width > 0 {
		args = append(args, "-x", strconv.Itoa(cfg.Width))
	}
	if cfg.Height > 0 {
		args = append(args, "-y", strconv.Itoa(cfg.Height))
	}

	if cfg.AlwaysOnTop {
		args = append(args, "-alwaysontop")
	}
	if cfg.Borderless {
		args = append(args, "-noborder")
	}
	if cfg.Fullscreen {
		args = append(args, "-fs")
	}

	args = append(args, videoPath)

	cmd := exec.Command(ffplayPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
		//CreationFlags: syscall.CREATE_NO_WINDOW,
		//HideWindow:    true, // optional
	}

	return cmd, nil
}
