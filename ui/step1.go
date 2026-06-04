package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gomp3_gui/app"
	"gomp3_gui/service"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// =======================
// Step1 UI
// =======================
type Step1UI struct {
	state     *app.AppState
	Container fyne.CanvasObject

	// UI components
	urlEntry   *widget.Entry
	messageBox *widget.Entry
	thumbImage *canvas.Image

	// unified status / progress
	statusLabel        *widget.Label
	progressBar        fyne.CanvasObject
	progressController *Progress
	downloadBar        *widget.ProgressBarInfinite
	playBar            *widget.ProgressBar
	progressContainer  *fyne.Container

	// Buttons
	downloadBtn *widget.Button
	playBtn     *widget.Button
	nextBtn     *widget.Button

	// Notice
	noticeUI *NoticeUI

	// play state
	isPlaying bool
	playCmd   *exec.Cmd
	onNext    func()

	// step4에 가기 위한 체크박스
	autoModeCheck *widget.Check
}

// =======================
// Step lifecycle
// =======================
func (ui *Step1UI) OnEnter() {
	if ui.state.VideoURL == "" {
		ui.ResetUI()
		return
	}
	ui.RefreshFromState()
}

func (ui *Step1UI) RefreshFromState() {
	if ui.urlEntry.Text != ui.state.VideoURL {
		ui.urlEntry.SetText(ui.state.VideoURL)
	}
	//ui.updateDownloadButton()
}

func (ui *Step1UI) ResetUI() {
	ui.urlEntry.SetText("")

	if ui.thumbImage != nil {
		noImage := filepath.Join(ui.state.Paths.Assets, "no_image.png")
		ui.thumbImage.File = noImage
		ui.thumbImage.Refresh()
	}

	ui.state.VideoURL = ""
	ui.state.Step1 = &app.Step1Result{}
	ui.state.Step2 = &app.Step2Result{}
	ui.state.Step3 = &app.Step3Result{}

	if ui.messageBox != nil {
		ui.messageBox.SetText("")
	}

	ui.playBtn.Disable()
	ui.nextBtn.Disable()
	ui.downloadBtn.Disable()
}

func (ui *Step1UI) resetProgress() {
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
// Build
// =======================
func BuildStep1(state *app.AppState, onNext func()) *Step1UI {

	ui := &Step1UI{
		state:  state,
		onNext: onNext,
	}

	binDir := state.Paths.Bin

	ff, err := service.EnsureFFmpeg(binDir)
	if err != nil {
		panic(err)
	}
	ytdlp, err := service.EnsureYtDlp(binDir)
	if err != nil {
		panic(err)
	}

	state.Tools.FFmpeg = ff.FFmpeg
	state.Tools.FFplay = ff.FFplay
	state.Tools.FFprobe = ff.FFprobe
	state.Tools.YtDlp = ytdlp

	ui.build()
	return ui
}

// =======================
// UI build
// =======================
func (ui *Step1UI) build() {
	//fmt.Println("I18n is nil?", ui.state.I18n == nil)

	// 체크박스 생성
	ui.autoModeCheck = widget.NewCheck("", nil)

	// URL
	ui.urlEntry = widget.NewEntry()
	ui.urlEntry.SetPlaceHolder(ui.state.I18n.T("step1.input.url_placeholder"))
	ui.urlEntry.OnChanged = func(s string) {
		ui.state.VideoURL = strings.TrimSpace(s)

		if ui.state.VideoURL != "" {
			ui.downloadBtn.Enable()
		} else {
			ui.downloadBtn.Disable()
		}
	}

	ui.urlEntry.OnSubmitted = func(string) {
		ui.fetchMeta()
	}

	ui.downloadBtn = widget.NewButton("Download", ui.fetchMeta)
	ui.downloadBtn.Disable()

	urlRow := container.NewBorder(
		nil,
		nil,
		ui.autoModeCheck,
		ui.downloadBtn,
		ui.urlEntry,
	)

	// Thumbnail
	noImage := filepath.Join(ui.state.Paths.Assets, "no_image.png")
	ui.thumbImage = canvas.NewImageFromFile(noImage)
	ui.thumbImage.FillMode = canvas.ImageFillContain
	ui.thumbImage.SetMinSize(fyne.NewSize(710, 399))

	// Status / Progress
	ui.statusLabel = widget.NewLabel("")
	ui.downloadBar = widget.NewProgressBarInfinite()
	ui.playBar = widget.NewProgressBar()

	progressRow := func(label *widget.Label, bar fyne.CanvasObject) fyne.CanvasObject {
		labelWrap := container.NewGridWrap(
			fyne.NewSize(120, label.MinSize().Height),
			label,
		)
		return container.NewBorder(nil, nil, labelWrap, nil, bar)
	}

	ui.progressBar = ui.downloadBar
	ui.progressContainer = container.NewStack(
		progressRow(ui.statusLabel, ui.progressBar),
	)

	// Buttons
	ui.playBtn = widget.NewButton(ui.state.I18n.T("btn.play"), ui.togglePlay)
	ui.playBtn.Disable()

	ui.nextBtn = widget.NewButton(ui.state.I18n.T("btn.next"), func() {
		ui.onNext()
	})
	ui.nextBtn.Disable()

	// notice test
	ui.noticeUI = NewNoticeUI(
		ui.state.Paths.Data,
	)

	// MessageBox
	ui.messageBox = widget.NewMultiLineEntry()
	ui.messageBox.SetMinRowsVisible(4)
	ui.messageBox.OnChanged = func(string) {}
	ui.messageBox.OnSubmitted = func(string) {}

	// Layout
	top := container.NewVBox(
		widget.NewLabelWithStyle(
			ui.state.I18n.T("step1.title"),
			fyne.TextAlignCenter,
			fyne.TextStyle{Bold: true},
		),
		urlRow,
	)

	center := container.NewVBox(
		ui.thumbImage,
		ui.progressContainer,
		ui.playBtn,
	)

	bottom := container.NewVBox(
		ui.messageBox,
		ui.nextBtn,
		ui.noticeUI.Container(),
	)

	ui.Container = container.NewBorder(top, bottom, nil, nil, center)
}

// =======================
// Status helper
// =======================
func (ui *Step1UI) setStatus(text string, bar fyne.CanvasObject, p *Progress) {
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

// =======================
// Logic
// =======================
// fetchMeta: 메타 수집 + 썸네일 처리 + 다운로드 시작
func (ui *Step1UI) fetchMeta() {
	logFilePath := filepath.Join(ui.state.Paths.Temp, "event.log")
	err := os.WriteFile(logFilePath, []byte(""), 0644)
	if err != nil {
		fyne.Do(func() {
			ui.appendLog("❌ event.log init fail: " + err.Error())
		})
	}

	if ui.isPlaying {
		ui.stopVideo()
	}

	url := ui.state.VideoURL
	if url == "" {
		return
	}

	ui.state.Step1 = &app.Step1Result{}
	ui.state.Step2 = &app.Step2Result{}
	ui.state.Step3 = &app.Step3Result{}

	ui.appendLog(ui.state.I18n.T("step1.meta.checking"))
	ui.downloadBtn.Disable()
	ui.playBtn.Disable()
	ui.nextBtn.Disable()

	p := NewProgress(ui.downloadBar, ui.statusLabel)
	ui.setStatus(ui.state.I18n.T("step1.meta.loading"), ui.downloadBar, p)
	p.StartDownload()

	go func() {
		// 1️⃣ 메타 수집
		meta, err := service.FetchYTMeta(ui.state.Tools.YtDlp, url)
		if err != nil {
			fyne.Do(func() {
				ui.appendLog(ui.state.I18n.T("step1.meta.fail") + err.Error())
			})
			return
		}

		// 2️⃣ 썸네일 다운로드 및 JPEG 변환
		raw := filepath.Join(ui.state.Paths.Temp, "thumb_raw")
		jpg := filepath.Join(ui.state.Paths.Temp, "thumbnail.jpg")

		if err := service.DownloadFile(meta.Thumbnail, raw); err == nil {
			if err := service.NormalizeThumbnail(raw, jpg); err == nil {
				fyne.Do(func() {
					ui.thumbImage.File = jpg
					ui.thumbImage.Refresh()
				})
			}
		}

		// 3️⃣ UI 갱신 및 다운로드 시작
		fyne.Do(func() {
			ui.appendLog(ui.state.I18n.T("step1.meta.success"))
			ui.appendLog("🎵 " + meta.Artist + " - " + meta.Title)
			ui.appendLog(
				fmt.Sprintf(ui.state.I18n.T("step1.meta.duration"), service.FormatTimestamp(meta.Duration)))
			ui.state.Step1.Meta = meta
		})

		//ui.startDownload(url, meta)
		ui.startDownload(url)
	}()
}

// startDownload: 기존 ytdownload.go 사용
// func (ui *Step1UI) startDownload(url string, meta *service.YTMeta) {
func (ui *Step1UI) startDownload(url string) {
	logFilePath := filepath.Join(ui.state.Paths.Temp, "event.log")
	// ----------------------------
	// Holiday JSON 자동 생성
	// ----------------------------
	year := time.Now().Year()

	err := service.EnsureHolidayYear(year)
	if err != nil {
		ui.appendLog("holiday generation failed: " + err.Error())
	}

	// 미리보기(ffplay)가 video.mp4를 잡고 있으면 재다운로드 시 덮어쓰기가
	// [WinError 32]로 실패한다. 다운로드 전에 우리가 띄운 미리보기를 종료해 핸들 해제.
	// (UI 정리는 playVideo의 cmd.Wait 고루틴이 fyne.Do로 수행)
	if ui.playCmd != nil && ui.playCmd.Process != nil {
		_ = ui.playCmd.Process.Kill()
	}

	p := NewProgress(ui.downloadBar, ui.statusLabel)
	ui.setStatus("Downloading...", ui.downloadBar, p)
	p.StartDownload()

	go func() {
		video, err := service.DownloadVideo(
			ui.state.Tools.YtDlp,
			url,
			ui.state.Paths.Video,
			logFilePath,
			func(s string) {
				ui.appendDownloadLog(s)
			},
			func(percent float64) { // progress 콜백 (0~1)
				fyne.Do(func() { p.UpdateDownload(percent, "") })
			},
		)

		if err != nil {
			fyne.Do(func() {
				ui.appendLog(ui.state.I18n.T("step1.download.fail") + err.Error())
				ui.downloadBtn.Enable()
			})
			return
		}

		fyne.Do(func() {
			ui.state.Step1.VideoPath = video
			ui.statusLabel.SetText(ui.state.I18n.T("step1.download.success"))
			ui.resetProgress()
			ui.playBtn.Enable()
			ui.nextBtn.Enable()
			ui.downloadBtn.Enable()
			if ui.autoModeCheck != nil {
				ui.state.AutoMode = ui.autoModeCheck.Checked
			}
		})
	}()
}

// =======================
// Play Video
// =======================
func (ui *Step1UI) togglePlay() {
	if ui.isPlaying {
		ui.stopVideo()
		return
	}
	ui.playVideo()
}

func (ui *Step1UI) playVideo() {
	if ui.state.Step1.VideoPath == "" {
		return
	}

	dur, err := service.GetMediaDuration(
		ui.state.Tools.FFprobe,
		ui.state.Step1.VideoPath,
	)
	if err != nil || dur <= 0 {
		return
	}

	cfgPath := filepath.Join(ui.state.Paths.Conf, "play_video.json")
	cfg, _ := service.LoadPlayConfig(cfgPath)
	cmd, err := service.BuildFFplayCmd(
		ui.state.Tools.FFplay,
		ui.state.Step1.VideoPath,
		cfg,
	)
	if err != nil {
		return
	}

	p := NewProgress(ui.playBar, ui.statusLabel)
	ui.setStatus("00:00 / "+formatTime(dur), ui.playBar, p)
	p.StartPlay(dur)

	_ = cmd.Start()

	ui.playCmd = cmd
	ui.isPlaying = true
	ui.playBtn.SetText(ui.state.I18n.T("btn.stop"))
	ui.appendLog(ui.state.I18n.T("btn.play"))

	go func() {
		start := time.Now()
		for ui.isPlaying {
			elapsed := time.Since(start).Seconds()
			fyne.Do(func() {
				ui.statusLabel.SetText(
					formatTime(elapsed) + " / " + formatTime(dur),
				)
			})
			time.Sleep(500 * time.Millisecond)
		}
	}()

	go func() {
		_ = cmd.Wait()
		fyne.Do(func() {
			ui.stopVideo()
		})
	}()
}

func (ui *Step1UI) stopVideo() {
	// 이미 중지 상태면 아무 것도 하지 않음
	if !ui.isPlaying {
		return
	}

	ui.isPlaying = false

	// ffplay 종료 처리 (이미 종료된 경우라도 안전)
	if ui.playCmd != nil && ui.playCmd.Process != nil {
		_ = ui.playCmd.Process.Kill()
		ui.playCmd = nil
	}

	// progress 종료
	if ui.progressController != nil {
		ui.progressController.FinishPlay()
		ui.progressController = nil
	}

	// UI
	// ui.playBar.Stop()
	ui.statusLabel.SetText(ui.state.I18n.T("btn.stop"))
	AppendLog(ui.messageBox, ui.state.I18n.T("btn.stop"))

	ui.resetPlayUI()
}

func (ui *Step1UI) resetPlayUI() {
	fyne.Do(func() {
		ui.playBtn.SetText(ui.state.I18n.T("btn.play"))
		ui.resetProgress()
	})
}

func (ui *Step1UI) appendDownloadLog(line string) {
	fyne.Do(func() {
		lines := strings.Split(ui.messageBox.Text, "\n")
		if strings.HasPrefix(line, "[download]") {
			// 마지막 줄 덮어쓰기
			if len(lines) == 0 {
				lines = []string{line}
			} else {
				lines[len(lines)-1] = line
			}
		} else {
			// 일반 로그: 새 줄 추가
			lines = append(lines, line)
		}
		ui.messageBox.SetText(strings.Join(lines, "\n"))

		// 마지막 줄로 스크롤
		ui.messageBox.CursorRow = len(lines)
	})
}

func (ui *Step1UI) appendLog(msg string) {
	if ui.messageBox != nil {
		fyne.Do(func() { AppendLog(ui.messageBox, msg) })
	}
}
