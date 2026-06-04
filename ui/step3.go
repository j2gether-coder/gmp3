package ui

import (
	"path/filepath"
	"sort"

	"gomp3_gui/app"
	"gomp3_gui/service"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type Step3UI struct {
	state *app.AppState

	Container fyne.CanvasObject

	artistEntry *widget.Entry
	titleEntry  *widget.Entry

	image      *canvas.Image
	messageBox *widget.Entry

	covers     []string
	coverIndex int

	ruleGroup   *widget.RadioGroup
	filenameBox *fyne.Container
	saveBtn     *widget.Button
	renameBtn   *widget.Button
	tagSaved    bool
}

// =======================
// Step lifecycle
// =======================
func (ui *Step3UI) OnEnter() {
	ui.MsgClearUI()
	ui.RefreshFromState() // ⭐ 이 한 줄이 핵심
}

func (ui *Step3UI) MsgClearUI() {
	if ui.messageBox != nil {
		ui.messageBox.SetText("")
		ui.messageBox.Refresh()
	}
}

// =======================
// Refresh from state
// =======================
func (ui *Step3UI) RefreshFromState() {

	if ui.state.Step2 == nil {
		return
	}

	step2 := ui.state.Step2

	ui.artistEntry.SetText(step2.Artist)
	ui.titleEntry.SetText(step2.Title)

	ui.loadCovers()

	if len(ui.covers) > 0 {
		ui.image.File = ui.covers[0]
		ui.image.Refresh()
	} else if step2.CoverPath != "" {
		ui.image.File = step2.CoverPath
		ui.image.Refresh()
	}
}

// =======================
// Builder
// =======================
func BuildStep3(state *app.AppState, backCallback func()) *Step3UI {
	ui := &Step3UI{state: state}
	ui.buildUI(backCallback)
	return ui
}

// =======================
// UI
// =======================
func (ui *Step3UI) buildUI(backCallback func()) {

	ui.artistEntry = widget.NewEntry()
	ui.titleEntry = widget.NewEntry()

	metaRow := func(label string, entry *widget.Entry) *fyne.Container {
		l := widget.NewLabel(label)
		labelWrap := container.NewGridWrap(
			fyne.NewSize(80, l.MinSize().Height),
			l,
		)
		return container.NewBorder(nil, nil, labelWrap, nil, entry)
	}

	ui.image = canvas.NewImageFromFile("")
	ui.image.FillMode = canvas.ImageFillContain
	ui.image.SetMinSize(fyne.NewSize(300, 300)) // 360

	ui.messageBox = widget.NewMultiLineEntry()
	//ui.messageBox.SetText("")
	ui.messageBox.SetMinRowsVisible(3)
	//ui.messageBox.Disable()
	ui.messageBox.OnChanged = func(string) {}
	ui.messageBox.OnSubmitted = func(string) {}

	prevBtn := widget.NewButton(ui.state.I18n.T("btn.prev"), ui.prevCover)
	nextBtn := widget.NewButton(ui.state.I18n.T("btn.next"), ui.nextCover)

	ui.saveBtn = widget.NewButton("💾 Save Tags", ui.saveTag)
	saveBtn := ui.saveBtn

	ui.ruleGroup = widget.NewRadioGroup(
		[]string{"No Changes", "Artist - Title", "Title - Artist"},
		nil,
	)
	ui.ruleGroup.Horizontal = true
	ui.ruleGroup.Disable()
	ui.ruleGroup.SetSelected("No Changes")
	ruleGroupContainer := container.NewCenter(ui.ruleGroup)

	ui.renameBtn = widget.NewButton("✏ Rename File", ui.applyRename)
	ui.renameBtn.Disable()

	ui.filenameBox = container.NewVBox(
		widget.NewSeparator(),
		widget.NewLabel("Filename Rule"),
		ruleGroupContainer,
		ui.renameBtn,
	)

	backBtn := widget.NewButton(ui.state.I18n.T("btn.first"), backCallback)
	exitBtn := widget.NewButton(ui.state.I18n.T("btn.exit"), func() {
		fyne.CurrentApp().Quit()
	})

	top := container.NewVBox(
		widget.NewLabelWithStyle(
			ui.state.I18n.T("step3.title"),
			fyne.TextAlignCenter,
			fyne.TextStyle{Bold: true},
		),
		metaRow("Artist", ui.artistEntry),
		metaRow("Title", ui.titleEntry),
	)

	// ======== 이미지 + 좌우 버튼 개선 ========
	// 이미지 좌우 버튼을 이미지 높이에 맞게 배치
	imageWithButtons := container.NewBorder(nil, nil, prevBtn, nextBtn, ui.image)
	center := container.NewVBox(
		imageWithButtons,
		saveBtn,
		ui.filenameBox,
	)

	bottom := container.NewVBox(
		ui.messageBox,
		backBtn,
		exitBtn,
	)

	ui.Container = container.NewBorder(top, bottom, nil, nil, center)

}

// =======================
// Cover
// =======================
func (ui *Step3UI) loadCovers() {
	ui.covers = nil

	pattern := filepath.Join(ui.state.Paths.Image, "cover_*.jpg")
	list, _ := filepath.Glob(pattern)

	sort.Strings(list)
	ui.covers = list
	ui.coverIndex = 0
}

func (ui *Step3UI) prevCover() {
	if len(ui.covers) == 0 {
		return
	}
	ui.coverIndex = (ui.coverIndex - 1 + len(ui.covers)) % len(ui.covers)
	ui.image.File = ui.covers[ui.coverIndex]
	ui.image.Refresh()
}

func (ui *Step3UI) nextCover() {
	if len(ui.covers) == 0 {
		return
	}
	ui.coverIndex = (ui.coverIndex + 1) % len(ui.covers)
	ui.image.File = ui.covers[ui.coverIndex]
	ui.image.Refresh()
}

// =======================
// Tag Save
// =======================
func (ui *Step3UI) saveTag() {
	step2 := ui.state.Step2
	meta := ui.state.Step1.Meta
	confPath := filepath.Join(ui.state.Paths.Conf, "cover.json")

	if step2 == nil || step2.MP3Path == "" {
		ui.appendLog("❌ No MP3.")
		return
	}

	if len(ui.covers) == 0 {
		ui.appendLog("❌ No selectable cover image.")
		return
	}

	// --- 입력값은 UI 스레드에서 캡처 (goroutine에서 ui/state 동시 접근 방지)
	selectedCover := ui.covers[ui.coverIndex]
	finalCover := filepath.Join(ui.state.Paths.Image, "cover_999.jpg")
	mp3Path := step2.MP3Path
	artist := step2.Artist
	title := step2.Title
	album := meta.Album
	year := meta.UploadDate[:4]
	sourceURL := ui.state.Step1.Meta.WebpageURL

	// 대용량 MP3 재작성(tag.Save)은 UI 스레드에서 수 초~수십 초 블로킹되어
	// "응답없음" 및 종료 시 크래시를 유발한다. 반드시 백그라운드에서 처리.
	ui.saveBtn.Disable()
	ui.appendLog("🎵 Tagging in progress...")

	go func() {
		if err := service.RenderCoverText(
			selectedCover,
			finalCover,
			confPath,
			service.CoverTextOption{
				Artist: artist,
				Title:  title,
				Size:   500,
			},
		); err != nil {
			fyne.Do(func() {
				ui.appendLog("❌ Faild to generate cover.")
				ui.saveBtn.Enable()
			})
			return
		}

		// 1) 텍스트 태그 적용 (가벼움 — 보통 padding 안에 들어감)
		if err := service.ApplyTextTags(
			mp3Path,
			artist,
			title,
			album,
			year,
			1,
			sourceURL,
		); err != nil {
			fyne.Do(func() {
				ui.appendLog("❌ Failed to apply text tags: " + err.Error())
				ui.saveBtn.Enable()
			})
			return
		}
		fyne.Do(func() {
			ui.appendLog("✅ Text tags applied")
		})

		// 2) 커버 이미지 적용 (MP3 전체 재작성 — 실패해도 텍스트 태그는 보존)
		if err := service.ApplyCoverImage(mp3Path, finalCover); err != nil {
			fyne.Do(func() {
				ui.appendLog("❌ Failed to apply cover image: " + err.Error())
				ui.saveBtn.Enable()
			})
			return
		}

		fyne.Do(func() {
			ui.tagSaved = true
			ui.ruleGroup.Enable()
			ui.renameBtn.Enable()
			ui.saveBtn.Enable()
			ui.appendLog(ui.state.I18n.T("step3.tag.success"))
		})
	}()
}

// =======================
// Rename
// =======================
func mapRule(selected string) service.FileNameRule {
	switch selected {
	case "Artist - Title":
		return service.FileNameRuleArtistTitle
	case "Title - Artist":
		return service.FileNameRuleTitleArtist
	default:
		return service.FileNameRuleDate
	}
}

func (ui *Step3UI) applyRename() {
	step2 := ui.state.Step2
	if step2 == nil || step2.MP3Path == "" {
		ui.appendLog("❌ No MP3")
		return
	}

	rule := mapRule(ui.ruleGroup.Selected)

	newPath, err := service.RenameMP3(
		step2.MP3Path,
		rule,
		service.ArtistTitleMeta{
			Artist: step2.Artist,
			Title:  step2.Title,
		},
	)

	if err != nil {
		ui.appendLog(ui.state.I18n.T("step3.rename.fail") + err.Error())
		return
	}

	step2.MP3Path = newPath
	ui.appendLog(ui.state.I18n.T("step3.rename.success"))
}

// =======================
// Log Helper
// =======================
func (ui *Step3UI) appendLog(msg string) {
	if ui.messageBox != nil {
		fyne.Do(func() { AppendLog(ui.messageBox, msg) })
	}
}
