package app

import "gomp3_gui/util"

type AppState struct {
	Paths *util.AppPaths
	Tools *ToolPaths

	VideoURL string

	Step1 *Step1Result
	Step2 *Step2Result
	Step3 *Step3Result

	AutoMode bool

	Lang string
	I18n *util.LangManager
}

type ToolPaths struct {
	FFmpeg  string
	FFplay  string
	FFprobe string
	YtDlp   string
}

func NewAppState(paths *util.AppPaths) *AppState {
	lang := util.DetectLanguage()

	return &AppState{
		Paths: paths,
		Lang:  lang,
		I18n:  util.NewLangManager(lang, paths.Data),
	}
}
