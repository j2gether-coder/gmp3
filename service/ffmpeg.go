package service

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const ffmpegURL = "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip"

type FFmpegTools struct {
	FFmpeg  string
	FFplay  string
	FFprobe string
}

func EnsureFFmpeg(binDir string) (*FFmpegTools, error) {
	ffmpegExe := filepath.Join(binDir, "ffmpeg.exe")
	ffplayExe := filepath.Join(binDir, "ffplay.exe")
	ffprobeExe := filepath.Join(binDir, "ffprobe.exe")

	if FileExists(ffmpegExe) && FileExists(ffplayExe) && FileExists(ffprobeExe) {
		if ok, _ := checkFFmpeg(ffmpegExe); ok {
			return &FFmpegTools{
				FFmpeg:  ffmpegExe,
				FFplay:  ffplayExe,
				FFprobe: ffprobeExe,
			}, nil
		}
	}

	tmp := filepath.Join(os.TempDir(), "gomp3_ffmpeg")
	zipPath := filepath.Join(tmp, "ffmpeg.zip")
	extract := filepath.Join(tmp, "extract")

	os.MkdirAll(extract, 0755)
	defer os.RemoveAll(tmp)

	if err := ChunkDownload(ffmpegURL, zipPath); err != nil {
		return nil, err
	}

	if err := Unzip(zipPath, extract); err != nil {
		return nil, err
	}

	ffmpegSrc, ffplaySrc, ffprobeSrc, err := findFFmpegBinaries(extract)
	if err != nil {
		return nil, err
	}

	CopyFile(ffmpegSrc, ffmpegExe)
	CopyFile(ffplaySrc, ffplayExe)
	CopyFile(ffprobeSrc, ffprobeExe)

	return &FFmpegTools{
		FFmpeg:  ffmpegExe,
		FFplay:  ffplayExe,
		FFprobe: ffprobeExe,
	}, nil

}

func findFFmpegBinaries(root string) (string, string, string, error) {
	var ffmpegExe, ffplayExe, ffprobeExe string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		name := strings.ToLower(info.Name())
		if name == "ffmpeg.exe" {
			ffmpegExe = path
		}
		if name == "ffplay.exe" {
			ffplayExe = path
		}
		if name == "ffprobe.exe" {
			ffprobeExe = path
		}
		return nil
	})
	if err != nil {
		return "", "", "", err
	}

	if ffmpegExe == "" || ffplayExe == "" || ffprobeExe == "" {
		return "", "", "", errors.New("ffmpeg 실행 파일을 찾을 수 없습니다")
	}
	return ffmpegExe, ffplayExe, ffprobeExe, nil
}

func checkFFmpeg(path string) (bool, string) {
	cmd := exec.Command(path, "-version")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} // Windows 전용
	if err := cmd.Run(); err != nil {
		return false, ""
	}
	line := strings.Split(out.String(), "\n")[0]
	return strings.HasPrefix(line, "ffmpeg version"), line
}
