package util

import (
	"os"
	"path/filepath"
)

type AppPaths struct {
	Root string

	Bin    string
	Assets string
	Conf   string
	Data   string
	Image  string
	Temp   string
	Audio  string
	Video  string
}

// GetAppPaths
// - bin/gomp3w.exe → 배포 모드
// - go run         → 개발 모드
func GetAppPaths() (*AppPaths, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}

	exeDir := filepath.Dir(exe)
	exeName := filepath.Base(exe)

	// -----------------------------
	// 배포 모드: exe가 bin 안에 있음
	// -----------------------------
	if filepath.Base(exeDir) == "bin" && exeName != "go-build.exe" {
		root := filepath.Clean(filepath.Join(exeDir, ".."))

		paths := &AppPaths{
			Root:   root,
			Bin:    exeDir,
			Assets: filepath.Join(root, "var", "assets"),
			Conf:   filepath.Join(root, "var", "conf"),
			Data:   filepath.Join(root, "var", "data"),
			Image:  filepath.Join(root, "var", "image"),
			Temp:   filepath.Join(root, "var", "temp"),
			Audio:  filepath.Join(root, "audio"),
			Video:  filepath.Join(root, "video"),
		}

		return paths, EnsureDirs(paths)
	}

	// -----------------------------
	// 개발 모드 (go run)
	// -----------------------------
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	paths := &AppPaths{
		Root:   wd,
		Bin:    filepath.Join(wd, "bin"),
		Assets: filepath.Join(wd, "var", "assets"),
		Conf:   filepath.Join(wd, "var", "conf"),
		Data:   filepath.Join(wd, "var", "data"),
		Image:  filepath.Join(wd, "var", "image"),
		Temp:   filepath.Join(wd, "var", "temp"),
		Audio:  filepath.Join(wd, "audio"),
		Video:  filepath.Join(wd, "video"),
	}

	return paths, EnsureDirs(paths)
}

// EnsureDirs
// - bin은 설치/빌드 영역이므로 생성하지 않음
func EnsureDirs(p *AppPaths) error {
	dirs := []string{
		p.Assets,
		p.Conf,
		p.Data,
		p.Image,
		p.Temp,
		p.Audio,
		p.Video,
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}
