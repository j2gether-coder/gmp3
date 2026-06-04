package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"gomp3_gui/app"
	"gomp3_gui/service"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// =======================
// Step2 UI
// =======================
type Step2UI struct {
	state     *app.AppState
	Container *fyne.Container

	ffmpegPath  string
	ffplayPath  string
	ffprobePath string
	outputDir   string

	artistEntry *widget.Entry
	titleEntry  *widget.Entry
	metaForm    *fyne.Container
	thumbImage  *canvas.Image

	bitrate        string
	CoverPath      string
	CoverGenerated bool
	bgMode         service.BgMode

	coverBtn   *widget.Button
	convertBtn *widget.Button
	playBtn    *widget.Button
	nextBtn    *widget.Button

	// unified status / progress
	statusLabel        *widget.Label
	progressBar        fyne.CanvasObject
	progressController *Progress
	downloadBar        *widget.ProgressBarInfinite
	playBar            *widget.ProgressBar
	progressContainer  *fyne.Container

	isPlaying bool
	playMu    sync.Mutex
	playCmd   *exec.Cmd

	messageBox *widget.Entry
	onNext     func()
}

// =======================
// Builder
// =======================
func BuildStep2(
	state *app.AppState,
	ffmpegPath string,
	ffplayPath string,
	ffprobePath string,
	outputDir string,
	onNext func(),
) *Step2UI {

	if state.Step2 == nil {
		state.Step2 = &app.Step2Result{}
	}

	ui := &Step2UI{
		state:       state,
		ffmpegPath:  ffmpegPath,
		ffplayPath:  ffplayPath,
		ffprobePath: ffprobePath,
		outputDir:   outputDir,
		onNext:      onNext,
	}

	ui.build()
	ui.RefreshFromState()
	return ui
}

// =======================
// State Refresh
// =======================
func (ui *Step2UI) OnEnter() {
	ui.clearUI()
	ui.RefreshFromState()
}

func (ui *Step2UI) clearUI() {
	if ui.artistEntry != nil {
		ui.artistEntry.SetText("")
	}
	if ui.titleEntry != nil {
		ui.titleEntry.SetText("")
	}
	if ui.thumbImage != nil {
		ui.thumbImage.File = ""
		ui.thumbImage.Refresh()
	}
	if ui.messageBox != nil {
		ui.messageBox.SetText("")
		ui.messageBox.Refresh()
	}
}

func (ui *Step2UI) RefreshFromState() {
	// 1. Meta 우선순위
	if ui.state.Step2 != nil &&
		(ui.state.Step2.Artist != "" || ui.state.Step2.Title != "") {
		ui.artistEntry.SetText(ui.state.Step2.Artist)
		ui.titleEntry.SetText(ui.state.Step2.Title)
	} else if ui.state.Step1 != nil && ui.state.Step1.Meta != nil {
		ui.artistEntry.SetText(ui.state.Step1.Meta.Artist)
		ui.titleEntry.SetText(ui.state.Step1.Meta.Title)
	} else {
		ui.artistEntry.SetText("")
		ui.titleEntry.SetText("")
	}

	// 2. 썸네일 우선순위
	imgPath := filepath.Join(ui.state.Paths.Temp, "thumbnail.jpg")
	if ui.state.Step2 != nil &&
		ui.state.Step2.CoverPath != "" &&
		ui.state.Step2.CoverGenerated {
		imgPath = ui.state.Step2.CoverPath
	}

	if _, err := os.Stat(imgPath); err == nil {
		ui.thumbImage.File = imgPath
		ui.thumbImage.Refresh()
	} else {
		fmt.Println("[Step2] image not found:", imgPath)
	}
}

// =======================
// UI Build
// =======================
func (ui *Step2UI) build() {
	// --- Meta Input & Modify
	var artist, title string
	if ui.state.Step1 != nil && ui.state.Step1.Meta != nil {
		artist = ui.state.Step1.Meta.Artist
		title = ui.state.Step1.Meta.Title
	}

	ui.artistEntry = widget.NewEntry()
	ui.artistEntry.SetText(artist)
	ui.titleEntry = widget.NewEntry()
	ui.titleEntry.SetText(title)

	// --- Meta input form
	metaRow := func(label string, entry *widget.Entry) *fyne.Container {
		l := widget.NewLabel(label)
		labelWrap := container.NewGridWrap(fyne.NewSize(80, l.MinSize().Height), l)
		return container.NewBorder(nil, nil, labelWrap, nil, entry)
	}
	titleLabel := widget.NewLabel(ui.state.I18n.T("step2.notice.edit_artist_title"))
	ui.metaForm = container.NewVBox(
		titleLabel,
		metaRow("Artist", ui.artistEntry),
		metaRow("Title", ui.titleEntry),
	)

	// --- Cover / Bitrate Radio
	bgSelectRadio := widget.NewRadioGroup(
		[]string{"그라이데이션(Gradient)", "평균색(AverageColor)", "검은색(Black)", "배경확장(BlueExtend)"},
		func(s string) {
			switch s {
			case "그라이데이션(Gradient)":
				ui.bgMode = service.BgAverageColorGradient
			case "평균색(AverageColor)":
				ui.bgMode = service.BgAverageColor
			case "검은색(Black)":
				ui.bgMode = service.BgBlack
			case "배경확장(BlueExtend)":
				ui.bgMode = service.BgBlurExtend
			}
		},
	)
	bgSelectRadio.Horizontal = true
	bgSelectRadio.SetSelected("그라이데이션(Gradient)")
	bgSelectRadioContainer := container.NewCenter(bgSelectRadio)

	bitrateRadio := widget.NewRadioGroup(
		[]string{"128 kbps", "192 kbps", "320 kbps"},
		func(s string) {
			switch s {
			case "128 kbps":
				ui.bitrate = "128k"
			case "192 kbps":
				ui.bitrate = "192k"
			case "320 kbps":
				ui.bitrate = "320k"
			}
		},
	)
	bitrateRadio.Horizontal = true
	bitrateRadio.SetSelected("128 kbps")
	bitrateRadioContainer := container.NewCenter(bitrateRadio)

	// --- Thumbnail Image ---
	coverPath := filepath.Join(ui.state.Paths.Assets, "no_image.png")
	ui.thumbImage = canvas.NewImageFromFile(coverPath)
	ui.thumbImage.SetMinSize(fyne.NewSize(300, 300))
	ui.thumbImage.FillMode = canvas.ImageFillContain

	// --- Buttons ---
	ui.coverBtn = widget.NewButton(ui.state.I18n.T("step2.cover.btn"), ui.generateCover)
	ui.convertBtn = widget.NewButton(ui.state.I18n.T("step2.convert.btn"), ui.startConvert)
	ui.convertBtn.Importance = widget.HighImportance
	ui.convertBtn.Disable()
	ui.playBtn = widget.NewButton(ui.state.I18n.T("btn.play"), ui.togglePlay)
	ui.playBtn.Disable()
	ui.nextBtn = widget.NewButton(ui.state.I18n.T("btn.next"), ui.onNext)
	ui.nextBtn.Disable()

	// --- Status / Progress
	ui.statusLabel = widget.NewLabel("■ Ready")
	ui.downloadBar = widget.NewProgressBarInfinite()
	ui.downloadBar.Hide()
	ui.progressBar = ui.downloadBar
	ui.playBar = widget.NewProgressBar()
	ui.progressController = nil

	progressRow := func(label *widget.Label, bar fyne.CanvasObject) fyne.CanvasObject {
		labelWrap := container.NewGridWrap(fyne.NewSize(100, label.MinSize().Height), label)
		return container.NewBorder(nil, nil, labelWrap, nil, bar)
	}
	ui.progressContainer = container.NewStack(progressRow(ui.statusLabel, ui.progressBar))

	// messageBox
	ui.messageBox = widget.NewMultiLineEntry()
	ui.messageBox.SetMinRowsVisible(3)
	ui.messageBox.OnChanged = func(string) {}
	ui.messageBox.OnSubmitted = func(string) {}
	//ui.messageBox.Disable()

	// --- Layout ---
	top := container.NewVBox(
		widget.NewLabelWithStyle(
			ui.state.I18n.T("step2.title"),
			fyne.TextAlignCenter,
			fyne.TextStyle{Bold: true}),
		ui.metaForm,
	)
	center := container.NewVBox(
		ui.thumbImage,
		bgSelectRadioContainer,
		ui.coverBtn,
		bitrateRadioContainer,
		ui.convertBtn,
		ui.progressContainer,
		ui.playBtn,
	)
	bottom := container.NewVBox(
		ui.messageBox,
		ui.nextBtn,
	)
	ui.Container = container.NewBorder(top, bottom, nil, nil, center)
}

// =======================
// Cover generation
// =======================
func (ui *Step2UI) generateCover() {
	artist := ui.artistEntry.Text
	title := ui.titleEntry.Text

	if artist == "" || title == "" {
		ui.appendLog(ui.state.I18n.T("step2.notice.edit_artist_title"))
		return
	}

	src := filepath.Join(ui.state.Paths.Temp, "thumbnail.jpg")
	dst := filepath.Join(ui.state.Paths.Image, "cover_000.jpg")
	ui.appendLog(ui.state.I18n.T("step2.cover.start"))
	ui.setStatus(ui.state.I18n.T("step2.cover.start"), ui.downloadBar, nil)
	ui.downloadBar.Start()

	go func() {
		err := service.ResizeThumbnailToCover(src, dst, 500, ui.bgMode)
		fyne.Do(func() {
			ui.downloadBar.Stop()
			if err != nil {
				ui.appendLog(ui.state.I18n.T("step2.cover.fail") + err.Error() + "\n")
				ui.setStatus("Cover image failed", ui.downloadBar, nil)
				return
			}
			ui.state.Step2.Artist = artist
			ui.state.Step2.Title = title
			ui.state.Step2.CoverPath = dst
			ui.state.Step2.CoverGenerated = true
			ui.state.Step2.Bitrate = ui.bitrate

			ui.RefreshFromState()
			ui.appendLog(ui.state.I18n.T("step2.cover.success"))
			ui.convertBtn.Enable()
			ui.playBtn.Disable()
			ui.nextBtn.Disable()
			ui.setStatus("Cover image created", nil, nil)
		})
	}()
}

// =======================
// MP3 Convert
// =======================
func (ui *Step2UI) startConvert() {
	logFilePath := filepath.Join(ui.state.Paths.Temp, "event.log")
	ui.state.Step2.Artist = ui.artistEntry.Text
	ui.state.Step2.Title = ui.titleEntry.Text
	ui.state.Step2.Bitrate = ui.bitrate

	videoPath := ui.state.Step1.VideoPath
	if videoPath == "" {
		ui.appendLog("❌ No videoPath")
		return
	}

	ui.convertBtn.Disable()
	ui.setStatus(ui.state.I18n.T("step2.convert.start"), ui.downloadBar, nil)
	ui.downloadBar.Start()
	ui.appendLog(ui.state.I18n.T("step2.convert.start"))

	// --- 진행률 라벨용 전체 길이 (예: 08:30:31)
	totalStr := "--:--:--"
	if dur, derr := service.GetMediaDuration(ui.ffprobePath, videoPath); derr == nil && dur > 0 {
		totalStr = service.FormatTimestamp(int(dur + 0.5))
	}

	// --- 진행 위치 콜백: ffmpeg 진행을 받아 progressInterval 간격으로만 라벨 갱신 (요건 6: 경량)
	const progressInterval = 30 * time.Second
	var (
		progressMu sync.Mutex
		lastUpdate time.Time
	)
	onProgress := func(currentSec float64) {
		progressMu.Lock()
		if !lastUpdate.IsZero() && time.Since(lastUpdate) < progressInterval {
			progressMu.Unlock()
			return
		}
		lastUpdate = time.Now()
		progressMu.Unlock()

		// 형식: 변환 중: 01:00:00 (08:00:00)
		label := fmt.Sprintf("변환 중...: %s (%s)",
			service.FormatTimestamp(int(currentSec+0.5)), totalStr)
		fyne.Do(func() { ui.statusLabel.SetText(label) })
	}

	go func() {
		mp3Path, err := service.ConvertVideoToMP3(
			ui.ffmpegPath, videoPath, ui.outputDir, ui.bitrate,
			false, // numbered=false, step2는 년월일시분
			0,     // index=0
			0,     // startSec=0
			0,     // durationSec=0
			logFilePath,
			onProgress,
		)

		if err != nil {
			fyne.Do(func() {
				ui.appendLog(ui.state.I18n.T("step2.convert.Fail") + err.Error() + "\n")
				ui.convertBtn.Enable()
				ui.playBtn.Disable()
				ui.nextBtn.Disable()
				ui.setStatus("MP3 Conversion failed.", nil, nil)
			})

			return
		}

		fyne.Do(func() {
			ui.state.Step2.MP3Path = mp3Path
			ui.appendLog(ui.state.I18n.T("step2.convert.success"))
			ui.appendLog("📁 " + mp3Path + "\n")
			ui.playBtn.Enable()
			ui.nextBtn.Enable()
			ui.setStatus("MP3 conversion complete.", nil, nil)
		})
	}()
}

// =======================
// 재생 관련
// =======================
func (ui *Step2UI) togglePlay() {
	if ui.isPlaying {
		ui.stopMP3()
	} else {
		ui.playMP3()
	}
}

func (ui *Step2UI) playMP3() {
	logFilePath := filepath.Join(ui.state.Paths.Temp, "event.log")
	ui.playMu.Lock()
	defer ui.playMu.Unlock()

	mp3Path := ui.state.Step2.MP3Path
	if mp3Path == "" {
		ui.appendLog("❌ No MP3 to play")
		return
	}

	if ui.state.Tools == nil || ui.state.Tools.FFplay == "" {
		ui.appendLog("❌ FFplay path not set.")
		return
	}

	dur, err := service.GetMediaDuration(
		ui.state.Tools.FFprobe,
		ui.state.Step1.VideoPath,
	)

	if err != nil || dur <= 0 {
		return
	}

	// 1️⃣ 미디어 전체 길이 가져오기 (media_probe.go)
	p := NewProgress(ui.playBar, ui.statusLabel)
	ui.setStatus("00:00 / "+formatTime(dur), ui.playBar, p)
	p.StartPlay(dur)

	dur, err = service.GetMediaDuration(ui.state.Tools.FFprobe, mp3Path)
	if err != nil {
		ui.appendLog("❌ Unable to get media duration: " + err.Error() + "\n")
		dur = 0
	}

	// 2️⃣ Progress 초기화
	ui.progressController = NewProgress(ui.playBar, ui.statusLabel)
	ui.progressController.StartPlay(dur)
	ui.progressBar = ui.playBar
	ui.playBar.SetValue(0)
	ui.statusLabel.SetText("▶ Ready...")

	ui.isPlaying = true
	ui.playBtn.SetText(ui.state.I18n.T("btn.stop"))

	// 3️⃣ 재생 시간 업데이트 (ticker 방식, Step1과 동일)
	go func() {
		start := time.Now()
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for ui.isPlaying {
			elapsed := time.Since(start).Seconds()

			// 🔹 UI 업데이트는 fyne.Do 안에서
			fyne.Do(func() {
				ui.progressController.UpdatePlay(elapsed)
			})

			if elapsed >= dur {
				break
			}
			<-ticker.C
		}

		// 재생 완료 UI 처리
		fyne.Do(func() {
			if ui.progressController != nil {
				ui.progressController.FinishPlay()
			}
			ui.isPlaying = false
			ui.playBtn.SetText(ui.state.I18n.T("btn.play"))
		})
	}()

	// 4️⃣ FFplay 재생 (로그만 처리, progress/ticker는 별도로)
	cmd, err := service.PlayAudio(
		ui.state.Tools.FFplay,
		logFilePath,
		mp3Path,
		func(msg string) { // onLog
			fyne.Do(func() { ui.appendLog(msg) })
		},
		nil, // onProgress → ticker에서 처리
		func(state string) { // onState
			fyne.Do(func() {
				ui.statusLabel.SetText(state)
				if state == "■ Stop" {
					ui.isPlaying = false
					ui.playBtn.SetText("▶ Play")
					ui.playBar.SetValue(1)
				}
			})
		},
	)
	if err != nil {
		ui.appendLog("❌ Play fail: " + err.Error())
		ui.isPlaying = false
		ui.playBtn.SetText("▶ Play")
		return
	}

	ui.playCmd = cmd
}

func (ui *Step2UI) stopMP3() {
	ui.playMu.Lock()
	defer ui.playMu.Unlock()

	if ui.playCmd != nil && ui.playCmd.Process != nil {
		_ = ui.playCmd.Process.Kill()
	}
	ui.playCmd = nil
	ui.isPlaying = false
	if ui.progressController != nil {
		ui.progressController.FinishPlay()
	}

	ui.playBtn.SetText("▶ Play")
	ui.playBar.SetValue(0)
	ui.statusLabel.SetText("■ Stop")
	ui.appendLog("⏹ Stop")
	ui.resetProgress()
}

// =======================
// Status Helper
// =======================
func (ui *Step2UI) setStatus(text string, bar fyne.CanvasObject, p *Progress) {
	fyne.Do(func() {
		ui.statusLabel.SetText(text)
		ui.progressController = p
		ui.progressBar = bar

		ui.progressContainer.Objects = []fyne.CanvasObject{
			container.NewBorder(
				nil, nil,
				container.NewGridWrap(
					fyne.NewSize(120, ui.statusLabel.MinSize().Height),
					ui.statusLabel,
				),
				nil,
				bar,
			),
		}
		ui.progressContainer.Refresh()
	})
}

func (ui *Step2UI) resetProgress() {
	if ui.progressBar == nil {
		return
	}

	// Progress controller 종료 (행위 기준)
	if ui.progressController != nil {
		if ui.isPlaying {
			ui.progressController.FinishPlay()
		} else {
			ui.progressController.FinishDownload()
		}
		ui.progressController = nil
	}

	// ProgressBar reset
	switch bar := ui.progressBar.(type) {
	case *widget.ProgressBarInfinite:
		bar.Stop()
	case *widget.ProgressBar:
		bar.SetValue(0)
	}

	ui.progressBar.Hide()
	ui.progressBar = nil
}

// =======================
// Log Helper
// =======================
func (ui *Step2UI) appendLog(msg string) {
	if ui.messageBox != nil {
		fyne.Do(func() { AppendLog(ui.messageBox, msg) })
	}
}
