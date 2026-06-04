package main

import (
	"log"
	"path/filepath"
	"runtime"

	"gomp3_gui/app"
	"gomp3_gui/ui"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
)

func main() {
	// =========================
	// App 생성 (크로스플랫폼)
	// =========================
	a := fyneapp.NewWithID("com.gomp3w.downloader")

	// 다크모드 강제
	a.Settings().SetTheme(theme.DarkTheme())

	// =========================
	// Window
	// =========================
	w := a.NewWindow("GoMP3w")
	w.Resize(fyne.NewSize(720, 820))
	w.CenterOnScreen()

	if runtime.GOOS == "windows" {
		w.SetMaster()
	}

	// =========================
	// 아이콘 (bundle 권장)
	// =========================

	iconPath := filepath.Join("gomp3.ico")
	iconRes, err := fyne.LoadResourceFromPath(iconPath)
	if err == nil {
		w.SetIcon(iconRes) // 창 아이콘 설정
	} else {
		log.Printf("아이콘 로드 실패: %v", err)
	}

	// =========================
	// AppState 초기화 (단일 책임)
	// =========================
	state := app.InitAppState()
	if state == nil {
		log.Fatal("AppState init fail")
	}

	// =========================
	// Step UI
	// =========================
	var (
		step1 *ui.Step1UI
		step2 *ui.Step2UI
		step3 *ui.Step3UI
		step4 *ui.Step4UI
	)

	step1 = ui.BuildStep1(
		state,
		func() {

			if state.AutoMode {
				w.SetContent(step4.Container)
				return
			}
			step2.OnEnter()
			w.SetContent(step2.Container)
		},
	)

	step2 = ui.BuildStep2(
		state,
		state.Tools.FFmpeg,
		state.Tools.FFplay,
		state.Tools.FFprobe,
		state.Paths.Audio,
		func() {
			step3.OnEnter()
			step3.RefreshFromState()
			w.SetContent(step3.Container)
		},
	)

	step3 = ui.BuildStep3(
		state,
		func() {
			// state 초기화
			state.VideoURL = ""
			state.Step1 = &app.Step1Result{}
			state.Step2 = &app.Step2Result{}
			state.Step3 = &app.Step3Result{}
			step1.OnEnter()
			w.SetContent(step1.Container)
		},
	)

	step4 = ui.BuildStep4(
		state,
		state.Tools.FFmpeg,
		func() {
			step1.OnEnter()
			w.SetContent(step1.Container)
		},
	)
	// =========================
	// Start
	// =========================
	w.SetContent(step1.Container)
	w.ShowAndRun()
}
