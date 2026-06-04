package app

import (
	"gomp3_gui/util"
	"log"
	"path/filepath"
)

// InitAppState
// 앱 시작 시 단 한 번 호출되는 초기화 진입점
func InitAppState() *AppState {

	paths, err := util.GetAppPaths()
	if err != nil {
		log.Fatal(err)
	}

	state := NewAppState(paths)

	state.Tools = &ToolPaths{
		FFmpeg:  filepath.Join(paths.Bin, "ffmpeg.exe"),
		FFplay:  filepath.Join(paths.Bin, "ffplay.exe"),
		FFprobe: filepath.Join(paths.Bin, "ffprobe.exe"),
		YtDlp:   filepath.Join(paths.Bin, "yt-dlp.exe"),
	}

	state.Step1 = &Step1Result{}
	state.Step2 = &Step2Result{}
	state.Step3 = &Step3Result{}

	return state
}
