// ui/setup4.go
package ui

import (
	"bufio"
	"fmt"
	"gomp3_gui/app"
	"gomp3_gui/service"
	"image/color"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type Step4UI struct {
	win fyne.Window

	// Radio / Mode
	modeRadio *widget.RadioGroup
	mode      string

	// State
	state     *app.AppState
	Container fyne.CanvasObject
	onBack    func()

	ffmpegPath string

	messageBox *widget.Entry

	// Button
	createTimestampBtn *widget.Button
	loadTimestampBtn   *widget.Button
	runBtn             *widget.Button
	currentStage       Step4Stage

	// User Timestamp Controls
	timestampSeparatorEntry   *widget.Entry
	refreshTimestampBtn       *widget.Button
	timestampUserControls     *fyne.Container
	timestampTableContainer   *fyne.Container
	timestampControlWrapper   *fyne.Container
	timestampPlaceholder      *fyne.Container
	timestampArtistFirstCheck *widget.Check
	mp3ProgressLabel          *widget.Label
	mp3ProgressContainer      *fyne.Container
	inputArtistFirst          bool
	randCover                 string

	// 분할모드 진행 알기
	currentProcessingRow int
	tableScroll          *container.Scroll
	rowHeight            float32
	processingActive     bool

	table          *widget.Table
	timestampTable *fyne.Container
	timestampData  []service.Segment
	segmentData    []SegmentItem
}

// Table 구조체
type SegmentItem struct {
	Index    int
	StartSec int
	EndSec   int
	Title    string
	Artist   string
}

const (
	ColIndex = iota
	ColTime
	ColTitle
	ColArtist
)

// DefaultSeparator: Title/Artist 구분자 기본값.
// 작성 안내문(createTimestampFile), 읽기(loadTimestampFile),
// 구분자 재적용(reloadUserTimestamp)에서 모두 이 값을 단일 기준으로 사용한다.
const DefaultSeparator = "-"

// Button 절차
type Step4Stage int

const (
	StageInit Step4Stage = iota + 1
	StageWriteDone
	StageReadDone
	StageProcessing
)

func BuildStep4(
	state *app.AppState,
	ffmpegPath string,
	onBack func(),
) *Step4UI {

	ui := &Step4UI{
		state:      state,
		ffmpegPath: ffmpegPath,
		onBack:     onBack,
	}

	ui.build()
	return ui
}

// UI
func (ui *Step4UI) build() {
	// ----------------------------
	// Title
	// ----------------------------
	title := widget.NewLabelWithStyle(
		ui.state.I18n.T("step4.title"),
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	// ----------------------------
	// messageBox
	// ----------------------------
	ui.messageBox = widget.NewMultiLineEntry()
	ui.messageBox.SetMinRowsVisible(6)

	// ----------------------------
	// 모드 선택
	// ----------------------------
	ui.modeRadio = widget.NewRadioGroup(
		[]string{"Chapter", "Timestamp", "Single"},
		func(value string) {
			ui.mode = strings.ToLower(value)
			ui.updateModeUI()
		},
	)

	ui.modeRadio.Horizontal = true
	modeRadioContainer := container.NewCenter(ui.modeRadio)

	modeRow := container.NewHBox(
		widget.NewLabel("Select MP3 Split Mode:"),
		modeRadioContainer,
	)

	// ----------------------------
	// Timestamp 버튼
	// ----------------------------
	ui.createTimestampBtn = widget.NewButton(
		"📝 Write Timestamp",
		ui.handleCreateTimestamp)
	ui.loadTimestampBtn = widget.NewButton(
		"📂 Read Timestamp",
		ui.loadTimestampFile)

	buttonRow := container.NewGridWithColumns(2,
		ui.createTimestampBtn,
		ui.loadTimestampBtn,
	)

	// ----------------------------
	// Timestamp Table
	// ----------------------------
	ui.segmentData = []SegmentItem{}
	ui.buildTable()

	headerLabel := widget.NewLabel("Timestamp Preview")
	refreshBtn := widget.NewButton("⟳ Refresh", func() {
		ui.setDefaultMode() // 모드 초기화
		ui.updateModeUI()   // UI 갱신
		ui.table.Refresh()  // 테이블 갱신
	})

	headerContainer := container.NewHBox(
		headerLabel,
		layout.NewSpacer(), // 라벨과 버튼 사이 여유
		refreshBtn,
	)

	// ----------------------------
	// Timestamp Controls
	// ----------------------------
	ui.timestampArtistFirstCheck = widget.NewCheck(
		"Input format is Artist - Title",
		func(bool) {
			// 순서(Artist/Title) 변경 시 즉시 미리보기 재파싱.
			// 아직 로드된 데이터가 없으면 무시(불필요한 에러 로그 방지).
			if len(ui.timestampData) > 0 {
				ui.reloadUserTimestamp(ui.currentSeparator())
			}
		},
	)
	ui.timestampArtistFirstCheck.SetChecked(false) // 기본값:  Title - Artist

	ui.timestampSeparatorEntry = widget.NewEntry()
	ui.timestampSeparatorEntry.SetText(DefaultSeparator) // 현재 구분자를 항상 보이게
	ui.timestampSeparatorEntry.SetPlaceHolder(DefaultSeparator)
	ui.refreshTimestampBtn = widget.NewButton(ui.state.I18n.T("btn.edit"), func() {
		ui.reloadUserTimestamp(ui.currentSeparator())
	})

	ui.timestampUserControls = container.NewHBox(
		widget.NewLabel("Title/Artist Separator:"),
		ui.timestampSeparatorEntry,
		ui.refreshTimestampBtn,
		ui.timestampArtistFirstCheck,
	)

	ui.timestampPlaceholder = container.NewHBox(
		widget.NewLabel(""),
	)

	ui.mp3ProgressLabel = widget.NewLabel("")

	ui.mp3ProgressContainer = container.NewBorder(
		nil,
		nil,
		widget.NewLabel("MP3 Conversion:"),
		nil,
		ui.mp3ProgressLabel,
	)
	ui.mp3ProgressLabel.Wrapping = fyne.TextWrapOff
	ui.mp3ProgressContainer.Hide()

	ui.timestampControlWrapper = container.NewVBox(
		ui.timestampUserControls,
		ui.mp3ProgressContainer,
	)

	// ----------------------------
	// 분할 모드 진행 알기
	// ----------------------------
	ui.rowHeight = 28
	ui.tableScroll = container.NewScroll(ui.table)
	ui.tableScroll.SetMinSize(fyne.NewSize(680, 300))

	// ----------------------------
	// 테이블 전체 영역
	// ----------------------------
	tableArea := container.NewBorder(
		headerContainer,
		ui.timestampControlWrapper,
		nil,
		nil,
		ui.tableScroll,
	)

	ui.timestampTable = tableArea

	// ----------------------------
	// 실행 / 이동 버튼
	// ----------------------------
	ui.runBtn = widget.NewButton(ui.state.I18n.T("btn.execute"), ui.run)

	backBtn := widget.NewButton(ui.state.I18n.T("btn.first"), func() {
		if ui.onBack != nil {
			ui.onBack()
		}
	})

	exitBtn := widget.NewButton(ui.state.I18n.T("btn.exit"), func() {
		fyne.CurrentApp().Quit()
	})

	// ----------------------------
	// 하단
	// ----------------------------
	bottomBar := container.NewGridWithColumns(2,
		backBtn,
		exitBtn,
	)

	// ----------------------------
	// 중앙
	// ----------------------------
	centerContent := container.NewVBox(
		modeRow,
		buttonRow,
		ui.timestampTable,
		ui.runBtn,
		ui.messageBox,
	)

	// ----------------------------
	// 전체 Layout
	// ----------------------------
	ui.Container = container.NewBorder(
		title,
		bottomBar,
		nil,
		nil,
		centerContent,
	)

	ui.setDefaultMode()
}

// ----------------------------
// 모드 변경 UI
// ----------------------------
func (ui *Step4UI) setDefaultMode() {
	meta := ui.state.Step1.Meta

	if meta != nil && len(meta.Chapters) > 0 {
		ui.modeRadio.SetSelected("Chapter") // callback 호출 X
		ui.mode = "chapter"
	} else {
		ui.modeRadio.SetSelected("Single") // callback 호출 X
		ui.mode = "single"
	}

	// 모드 UI 갱신: preview, 버튼 상태, table 초기화 등
	ui.updateModeUI()

	fyne.Do(func() {
		ui.table.Refresh()
	})
}

func (ui *Step4UI) updateModeUI() {
	if ui.runBtn == nil {
		return
	}
	ui.clearLog()
	ui.segmentData = []SegmentItem{}

	switch ui.mode {

	case "chapter":
		ui.timestampUserControls.Show()
		ui.timestampPlaceholder.Hide()
		ui.buildChapterPreview()

	case "timestamp":
		ui.timestampUserControls.Show()
		ui.timestampPlaceholder.Hide()
		ui.buildUserTimestampPreview()

	case "single":
		ui.timestampUserControls.Hide()
		ui.timestampPlaceholder.Show()
		ui.buildSinglePreview()
	}

	ui.currentStage = StageInit
	ui.applyStage()
}

// ----------------------------
// Pipeline 실행
// ----------------------------
func (ui *Step4UI) runPipeline() {
	video := ui.state.Step1.VideoPath
	meta := ui.state.Step1.Meta
	total := len(ui.segmentData)

	if video == "" || meta == nil || total == 0 {
		fyne.Do(func() {
			ui.processingActive = false
			ui.currentProcessingRow = -1
			ui.table.Refresh()
			ui.appendLog("❌ No Step1 results or segment data available.")
		})
		return
	}

	// =========================
	// 공통 준비
	// =========================
	logFilePath := filepath.Join(ui.state.Paths.Temp, "event.log")
	videoPath := ui.state.Step1.VideoPath
	outputDir := filepath.Join(ui.state.Paths.Audio)
	bitrate := "128k"

	// =========================
	// 랜덤 커버 1회 선택
	// =========================
	covers, err := filepath.Glob(filepath.Join(ui.state.Paths.Image, "cover_*.jpg"))
	if err != nil || len(covers) == 0 {
		fyne.Do(func() {
			ui.appendLog("❌ No cover candidates found")
			ui.runBtn.Enable()
		})
		return
	}

	// cover_000 / cover_999 제외
	var candidates []string
	for _, c := range covers {
		base := filepath.Base(c)
		if base == "cover_000.jpg" || base == "cover_999.jpg" {
			continue
		}
		candidates = append(candidates, c)
	}

	if len(candidates) == 0 {
		fyne.Do(func() {
			ui.appendLog("❌ No valid cover candidates available")
			ui.runBtn.Enable()
		})
		return
	}

	// 랜덤 선택
	rand.Seed(time.Now().UnixNano())
	ui.randCover = candidates[rand.Intn(len(candidates))]

	ui.mp3ProgressContainer.Show()
	ui.timestampUserControls.Hide()

	for i, seg := range ui.segmentData {
		index := i

		// 현재 처리 Row 강조
		fyne.Do(func() {
			ui.currentProcessingRow = i
			ui.table.Refresh()

			offsetY := float32(i) * ui.rowHeight
			ui.tableScroll.Offset = (fyne.NewPos(0, offsetY))
			ui.tableScroll.Refresh()
		})

		fyne.Do(func() {
			ui.appendLog(fmt.Sprintf("▶ %d/%d Conversion started: %s", i+1, total, seg.Title))
		})

		fyne.Do(func() {
			ui.mp3ProgressLabel.SetText(
				fmt.Sprintf("%d/%d (%.0f%%)", i+1, total,
					float64(index)/float64(total)*100),
			)
		})

		duration := seg.EndSec - seg.StartSec
		if duration <= 0 {
			ui.appendLog(fmt.Sprintf("⚠️ 잘못된 구간: start=%d end=%d", seg.StartSec, seg.EndSec))
			continue
		}

		// =========================
		// 1️⃣ MP3 변환
		// =========================
		mp3Path, err := service.ConvertVideoToMP3(
			ui.ffmpegPath,
			videoPath,
			outputDir,
			bitrate,
			true,
			index+1,
			seg.StartSec,
			duration,
			logFilePath,
			nil, // 진행률 콜백 미사용 (구간 변환은 짧음)
		)

		if err != nil {
			fyne.Do(func() {
				ui.processingActive = false
				ui.currentProcessingRow = -1
				ui.table.Refresh()
				ui.appendLog(fmt.Sprintf("❌ Segment %d Conversion failed: %s", index, err.Error()))
				//ui.runBtn.Enable()
			})
			return
		}

		fyne.Do(func() {
			ui.appendLog(fmt.Sprintf("✅ %d/%d Conversion completed: %s", i+1, total, mp3Path))
		})

		// =========================
		// 2️⃣ cover 생성 (1회만)
		// =========================
		finalCover, err := ui.generateFinalCover(&seg, ui.randCover)
		if err != nil {
			fyne.Do(func() {
				ui.processingActive = false
				ui.currentProcessingRow = -1
				ui.table.Refresh()
				ui.appendLog("❌ Generate cover_999 failed: " + err.Error())
			})
			return
		}

		fyne.Do(func() {
			ui.appendLog("✅ cover_999 created: " + finalCover)
		})

		// =========================
		// 3️⃣ MP3 태그 적용
		// =========================
		if err := ui.applyTags(mp3Path, seg.Index, seg.Title, seg.Artist, finalCover); err != nil {
			fyne.Do(func() {
				ui.appendLog(fmt.Sprintf("❌ Segment %d Tag Application failed: %s", seg.Index, err.Error()))
			})
			return
		}

		fyne.Do(func() {
			ui.appendLog(fmt.Sprintf("✅ Segment %d Tag applied successfully", seg.Index))
		})
	}

	// =========================
	// 완료
	// =========================
	fyne.Do(func() {
		ui.processingActive = false
		ui.currentProcessingRow = -1
		ui.mp3ProgressContainer.Hide()
		ui.timestampUserControls.Show()
		ui.table.Refresh()
		ui.appendLog("✅ MP3 conversion and Taggging completed")
		ui.currentStage = StageInit
		ui.applyStage()
	})
}

// ----------------------------
// 실행 버튼
// ----------------------------
func (ui *Step4UI) run() {
	if ui.state.Step1 == nil || ui.state.Step1.VideoPath == "" {
		ui.appendLog("❌ No Step1 results available.")
		return
	}

	if len(ui.segmentData) == 0 {
		ui.appendLog("❌ No segment data available")
		return
	}

	if ui.processingActive {
		ui.appendLog("⚠ Already running.")
		return
	}

	// 실행 버튼 비활성화 (무한 클릭 방지)
	ui.appendLog("🚀 Automatic processing started.")
	ui.processingActive = true
	ui.currentProcessingRow = -1
	ui.mp3ProgressLabel.SetText("0/0")

	ui.currentStage = StageProcessing
	ui.applyStage()
	go ui.runPipeline()
}

// --- UTIL
// createCover: thumbnail → cover_000.jpg 생성
func (ui *Step4UI) generateFinalCover(seg *SegmentItem, baseCover string) (string, error) {
	coverFinal := filepath.Join(ui.state.Paths.Image, "cover_999.jpg")
	confPath := filepath.Join(ui.state.Paths.Conf, "cover.json")

	var title, artist string
	if seg == nil {
		title = ui.state.Step1.Meta.Title
		artist = ui.state.Step1.Meta.Artist
	} else {
		title = seg.Title
		artist = seg.Artist
	}

	opt := service.CoverTextOption{
		Title:  title,
		Artist: artist,
		Size:   500,
	}

	err := service.RenderCoverText(
		baseCover,
		coverFinal,
		confPath,
		opt,
	)
	if err != nil {
		return "", err
	}

	return coverFinal, nil
}

// TAG
// 텍스트 태그 → 커버 이미지 순으로 분리 호출한다.
// 커버 이미지 적용은 MP3 전체를 재작성하므로 UAC/AV 환경에서
// 실패 확률이 높지만, 텍스트 태그는 그 전에 이미 저장된 상태가 된다.
func (ui *Step4UI) applyTags(
	mp3Path string,
	track int,
	title,
	artist string,
	coverPath string,
) error {
	album := ""
	year := ""
	sourceURL := ""

	if ui.state.Step1.Meta != nil {
		album = ui.state.Step1.Meta.Album
		if len(ui.state.Step1.Meta.UploadDate) >= 4 {
			year = ui.state.Step1.Meta.UploadDate[:4]
		}
		sourceURL = ui.state.Step1.Meta.WebpageURL
	}

	// 1) 텍스트 태그
	if err := service.ApplyTextTags(
		mp3Path,
		artist,
		title,
		album,
		year,
		track,
		sourceURL,
	); err != nil {
		return fmt.Errorf("Text tag application failed: %w", err)
	}
	ui.appendLog(fmt.Sprintf("✅ Text tag applied: %s (track %d)", mp3Path, track))

	// 2) 커버 이미지
	if coverPath == "" {
		return nil
	}
	if err := service.ApplyCoverImage(mp3Path, coverPath); err != nil {
		return fmt.Errorf("Cover image application failed: %w", err)
	}
	ui.appendLog(fmt.Sprintf("✅ Cover image applied: %s (track %d)", mp3Path, track))

	return nil
}

// ----------------------------
// Timestamp 파일 로드
// ----------------------------
func (ui *Step4UI) getTimestampPath() string {
	return filepath.Join(ui.state.Paths.Temp, "timestamp.txt")
}

func (ui *Step4UI) createTimestampFile() error {

	path := ui.getTimestampPath()

	content := `# [Tip] 동영상의 설명란이나 댓글에 있는 timestamp를 복사하여 붙여넣기 하세요.
# 형식은 다음과 같습니다.
# 00:00:00 Title Artist
# 또는
# 00:00:00 Artist Title
# ※ Artist와 Title의 구분자는 '-' 입니다.
# [예시]
# 00:00:00 Intro - ArtistName
# 00:03:12 SongTitle - ArtistName

`

	os.WriteFile(path, []byte(content), 0644)

	exec.Command("notepad", path).Start()

	ui.appendLog("📝 timestamp.txt 초기화 및 메모장 실행")

	return nil
}

func (ui *Step4UI) loadTimestampFile() {

	path := ui.getTimestampPath()

	err := ui.cleanTimestampFile(path)
	if err != nil {
		ui.appendLog("❌ timestamp refinement failed: " + err.Error())
		return
	}

	sep := ui.currentSeparator()

	inputArtistFirst := ui.timestampArtistFirstCheck.Checked

	items, err := service.ParseUserTimestampFile(
		path,
		ui.state.Step1.Meta.Duration,
		sep,
		ui.state.Step1.Meta.Artist,
		inputArtistFirst,
	)
	if err != nil {
		ui.appendLog("❌ Failed to parse timestamp: " + err.Error())
		return
	}

	ui.timestampData = items
	ui.buildUserTimestampPreview()

	if len(items) > 0 {
		ui.currentStage = StageReadDone
		ui.applyStage()
	}

	ui.appendLog("✅ timestamp loaded successfully")
}

func (ui *Step4UI) cleanTimestampFile(path string) error {

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")

	var cleaned []string

	for _, line := range lines {

		line = strings.TrimSpace(line)

		// 빈 줄 제거
		if line == "" {
			continue
		}

		// 주석 제거
		if strings.HasPrefix(line, "#") {
			continue
		}

		cleaned = append(cleaned, line)
	}

	// 마지막 줄만 개행 유지
	output := strings.Join(cleaned, "\n") + "\n"

	return os.WriteFile(path, []byte(output), 0644)
}

// ----------------------------
// Case 별 Table Preview
// ----------------------------
func (ui *Step4UI) buildChapterPreview() {

	meta := ui.state.Step1.Meta
	if meta == nil {
		return
	}

	ui.segmentData = []SegmentItem{}

	// Chapter가 있으면 사용
	if len(meta.Chapters) > 0 {

		for i, ch := range meta.Chapters {

			ui.segmentData = append(ui.segmentData, SegmentItem{
				Index:    i + 1,
				StartSec: ch.StartTime,
				EndSec:   ch.EndTime,
				Title:    ch.Title,
				Artist:   meta.Artist,
			})
		}

	} else {
		// fallback: 전체 영상 1개
		ui.segmentData = append(ui.segmentData, SegmentItem{
			Index:    1,
			StartSec: 0,
			EndSec:   meta.Duration,
			Title:    meta.Title,
			Artist:   meta.Artist,
		})
	}

	ui.table.Refresh()
}

func (ui *Step4UI) buildUserTimestampPreview() {

	ui.segmentData = []SegmentItem{}

	for i, ts := range ui.timestampData {
		ui.segmentData = append(ui.segmentData, SegmentItem{
			Index:    i + 1,
			StartSec: ts.StartSec,
			EndSec:   ts.EndSec,
			Title:    ts.Title,
			Artist:   ts.Artist,
		})
	}

	ui.table.Refresh()
}

func (ui *Step4UI) buildSinglePreview() {

	meta := ui.state.Step1.Meta
	if meta == nil {
		return
	}

	ui.segmentData = []SegmentItem{
		{
			Index:    1,
			StartSec: 0,
			EndSec:   meta.Duration,
			Title:    meta.Title,
			Artist:   meta.Artist,
		},
	}

	ui.table.Refresh()
}

func (ui *Step4UI) reloadUserTimestamp(sep string) {

	path := ui.getTimestampPath()
	if path == "" {
		ui.appendLog("❌ No timestamp file available.")
		return
	}

	totalDuration := ui.state.Step1.Meta.Duration
	fallbackArtist := ui.state.Step1.Meta.Artist // 또는 ChannelName 등

	inputArtistFirst := ui.timestampArtistFirstCheck.Checked

	items, err := service.ParseUserTimestampFile(
		path,
		totalDuration,
		sep,
		fallbackArtist,
		inputArtistFirst,
	)
	if err != nil {
		ui.appendLog("❌ timestamp parsing failed: " + err.Error())
		return
	}

	ui.timestampData = items
	ui.buildUserTimestampPreview()

	// 구분자/순서는 파싱 옵션이므로 입력값을 유지하고 버튼도 활성 상태로 둔다.
	// (몇 번이든 재적용 가능 — 구분자 재수정 불가 문제 해결)

	ui.appendLog(fmt.Sprintf(
		"✅ Separator '%s' applied successfully (fallback artist: %s)",
		sep,
		fallbackArtist,
	))
	ui.table.Refresh()
}

// =========================
// Table 생성
// =========================
func (ui *Step4UI) buildHeader() *fyne.Container {

	col0 := container.NewGridWrap(
		fyne.NewSize(50, 28),
		widget.NewLabelWithStyle("#", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)

	col1 := container.NewGridWrap(
		fyne.NewSize(180, 28),
		widget.NewLabelWithStyle("Time", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)

	col2 := container.NewGridWrap(
		fyne.NewSize(240, 28),
		widget.NewLabelWithStyle("Title", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)

	col3 := container.NewGridWrap(
		fyne.NewSize(120, 28),
		widget.NewLabelWithStyle("Artist", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)

	return container.NewHBox(col0, col1, col2, col3)
}

func (ui *Step4UI) buildTable() {

	ui.table = widget.NewTable(
		func() (int, int) {
			return len(ui.segmentData), 4
		},
		func() fyne.CanvasObject {
			bg := canvas.NewRectangle(color.Transparent)
			label := widget.NewLabel("")
			label.Truncation = fyne.TextTruncateEllipsis
			return container.NewMax(bg, label)
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {

			container := obj.(*fyne.Container)
			bg := container.Objects[0].(*canvas.Rectangle)
			label := container.Objects[1].(*widget.Label)

			// 범위 체크 (중요)
			if id.Row >= len(ui.segmentData) {
				label.SetText("")
				label.TextStyle = fyne.TextStyle{}
				return
			}

			row := ui.segmentData[id.Row]

			switch id.Col {
			case 0:
				label.SetText(strconv.Itoa(row.Index))
			case 1:
				label.SetText(fmt.Sprintf("%s ~ %s",
					service.FormatTimestamp(row.StartSec),
					service.FormatTimestamp(row.EndSec),
				))
			case 2:
				label.SetText(row.Title)
			case 3:
				label.SetText(row.Artist)
			}

			// 🔥 현재 처리중 Row 강조
			if ui.processingActive && id.Row == ui.currentProcessingRow && id.Col == 0 {
				label.TextStyle = fyne.TextStyle{Bold: true}

				bg.FillColor = theme.PrimaryColor()

			} else {
				label.TextStyle = fyne.TextStyle{}
				bg.FillColor = color.Transparent
			}
		},
	)

	ui.table.SetColumnWidth(0, 50)
	ui.table.SetColumnWidth(1, 180)
	ui.table.SetColumnWidth(2, 240)
	ui.table.SetColumnWidth(3, 120)
}

// =========================
// Chapter용 Timestamp File
// =========================
func (ui *Step4UI) WriteChapterToTimestampFile() error {

	filePath := ui.getTimestampPath()

	file, err := os.Create(filePath) // 무조건 덮어쓰기
	if err != nil {
		return fmt.Errorf("Failed to create timestamp file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	meta := ui.state.Step1.Meta

	for _, ch := range meta.Chapters {

		ts := service.FormatTimestamp(ch.StartTime)

		line := fmt.Sprintf("%s %s\n",
			ts,
			strings.TrimSpace(ch.Title),
		)

		if _, err := writer.WriteString(line); err != nil {
			return fmt.Errorf("Failed to write timestamp: %w", err)
		}
	}

	return writer.Flush()
}

// =========================
// Mode(chapter, user timestamp)별 분기
// =========================
func (ui *Step4UI) handleCreateTimestamp() {

	switch ui.mode {

	case "chapter":

		if err := ui.WriteChapterToTimestampFile(); err != nil {
			ui.appendLog("❌ Chapter → Failed to write timestamp.")
			return
		}
		ui.appendLog("✅ Chapter → timestamp.txt created successfully.")

	case "timestamp":

		// 🔥 기존 user 전용 함수 그대로 사용
		if err := ui.createTimestampFile(); err != nil {
			ui.appendLog("❌ Failed to write Timestamp.")
			return
		}
		ui.appendLog("✅ User Timestamp → timestamp.txt created successfully.")

	default:
		ui.appendLog("⚠ Timestamp writing is not available in the current mode.")
	}
	ui.currentStage = StageWriteDone
	ui.applyStage()
}

// =========================
// Clear Log
// =========================s
func (ui *Step4UI) clearLog() {
	fyne.Do(func() {
		ui.messageBox.SetText("")
		ui.messageBox.Refresh()
	})
}

// =======================
// Stage Button
// =======================
func (ui *Step4UI) applyStage() {

	// 기본값
	ui.createTimestampBtn.Disable()
	ui.loadTimestampBtn.Disable()
	ui.runBtn.Disable()

	ui.createTimestampBtn.Importance = widget.MediumImportance
	ui.loadTimestampBtn.Importance = widget.MediumImportance
	ui.runBtn.Importance = widget.MediumImportance

	if ui.mode == "single" {
		ui.runBtn.Enable()
		ui.runBtn.Importance = widget.HighImportance
		ui.createTimestampBtn.Refresh()
		ui.loadTimestampBtn.Refresh()
		ui.runBtn.Refresh()
		return
	}

	switch ui.currentStage {

	case StageInit:
		ui.createTimestampBtn.Enable()
		ui.createTimestampBtn.Importance = widget.HighImportance

	case StageWriteDone:
		ui.loadTimestampBtn.Enable()
		ui.loadTimestampBtn.Importance = widget.HighImportance

	case StageReadDone:
		ui.runBtn.Enable()
		ui.runBtn.Importance = widget.HighImportance

	case StageProcessing:
		// 전부 비활성화
	}

	ui.createTimestampBtn.Refresh()
	ui.loadTimestampBtn.Refresh()
	ui.runBtn.Refresh()
}

// =======================
// Helper
// =======================
// currentSeparator: 입력칸의 현재 구분자(공백 제거). 비어 있으면 기본값.
func (ui *Step4UI) currentSeparator() string {
	sep := strings.TrimSpace(ui.timestampSeparatorEntry.Text)
	if sep == "" {
		sep = DefaultSeparator
	}
	return sep
}

func (ui *Step4UI) appendLog(msg string) {
	if ui.messageBox != nil {
		fyne.Do(func() { AppendLog(ui.messageBox, msg) })
	}
}
