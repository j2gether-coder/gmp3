package ui

import (
	"encoding/json"
	"image/color"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gomp3_gui/service"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	noticeHeight   = 120
	scrollInterval = 5 * time.Second
)

// -------------------- DATA --------------------

type NoticeFile struct {
	Version int          `json:"version"`
	Items   []NoticeItem `json:"items"`
}

type NoticeItem struct {
	Type    string   `json:"type"`
	Content []string `json:"content"`
	URL     string   `json:"url"`
	Source  string   `json:"-"` // holiday or notice.json
}

// -------------------- CLICK LAYER --------------------

type ClickLayer struct {
	widget.BaseWidget
	onTap func()
}

func NewClickLayer(onTap func()) *ClickLayer {
	c := &ClickLayer{onTap: onTap}
	c.ExtendBaseWidget(c)
	return c
}

func (c *ClickLayer) Tapped(*fyne.PointEvent) {
	if c.onTap != nil {
		c.onTap()
	}
}

func (c *ClickLayer) TappedSecondary(*fyne.PointEvent) {}

func (c *ClickLayer) CreateRenderer() fyne.WidgetRenderer {
	rect := canvas.NewRectangle(color.Transparent)

	return &clickLayerRenderer{rect: rect}
}

type clickLayerRenderer struct {
	rect *canvas.Rectangle
}

func (r *clickLayerRenderer) Layout(size fyne.Size) {
	r.rect.Resize(size)
}

func (r *clickLayerRenderer) MinSize() fyne.Size {
	return fyne.NewSize(0, 0)
}

func (r *clickLayerRenderer) Refresh() {
	canvas.Refresh(r.rect)
}

func (r *clickLayerRenderer) Destroy() {}

func (r *clickLayerRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.rect}
}

// -------------------- UI --------------------

type NoticeUI struct {
	dataDir string
	items   []NoticeItem

	root            *fyne.Container
	selectedPattern string
	bgImage         *canvas.Image
	baseStack       *fyne.Container

	visibleStart int
	visibleLines int
	scrollTicker *time.Ticker
	currentLines []string

	currentItem *NoticeItem
	linesCont   *fyne.Container
	ticker      *time.Ticker
}

// -------------------- CONSTRUCTOR --------------------

func NewNoticeUI(dataDir string) *NoticeUI {

	ui := &NoticeUI{
		dataDir: dataDir,
	}

	// 텍스트 컨테이너
	ui.linesCont = container.NewVBox()

	centered := container.NewCenter(ui.linesCont)

	textContainer := container.NewMax(centered)

	// 클릭 레이어
	clickLayer := NewClickLayer(func() {
		if ui.currentItem != nil && ui.currentItem.URL != "" {
			openURL(ui.currentItem.URL)
		}
	})

	// 배경 패턴 교체(Notice Area)
	ui.selectedPattern = randomPattern(ui.dataDir)
	var background fyne.CanvasObject

	if ui.selectedPattern != "" {
		background = buildTiledBackground(ui.selectedPattern, 720, noticeHeight)
	} else {
		background = canvas.NewRectangle(color.Black) // fallback
	}

	// overlay := canvas.NewRectangle(
	// 	getTimeOverlayColor(time.Now()),
	// )
	// overlay.Resize(fyne.NewSize(720, noticeHeight))

	ui.baseStack = container.NewStack(
		background,
		//	overlay,
		textContainer,
	)

	stack := container.NewStack(
		ui.baseStack,
		clickLayer,
	)

	ui.root = container.New(
		layout.NewGridWrapLayout(fyne.NewSize(720, noticeHeight)),
		stack,
	)

	// 데이터 로드
	ui.loadItems()
	ui.pickRandomItem()

	// 🔥 한 줄 스크롤 ticker만 사용
	go func() {
		time.Sleep(scrollInterval)

		ui.scrollTicker = time.NewTicker(scrollInterval)
		for range ui.scrollTicker.C {
			fyne.Do(func() {
				ui.scrollOneLine()
			})
		}
	}()

	return ui
}

func (n *NoticeUI) Container() *fyne.Container {
	return n.root
}

// -------------------- LOAD DATA --------------------
func (n *NoticeUI) loadItems() {

	// notice.json
	path := filepath.Join(n.dataDir, "notice.json")
	data, err := os.ReadFile(path)
	if err == nil {
		var file NoticeFile
		if json.Unmarshal(data, &file) == nil {
			for _, item := range file.Items {
				if len(item.Content) > 0 {
					item.Source = "notice.json"
					n.items = append(n.items, item)
				}
			}
		}
	}

	// holiday
	if msg, err := service.BuildTodayNotice(time.Now()); err == nil && msg != "" {
		n.items = append(n.items, NoticeItem{
			Type:    "text",
			Content: []string{msg},
			Source:  "holiday",
		})
	}
}

// -------------------- RANDOM PICK --------------------
func (n *NoticeUI) pickRandomItem() {

	if len(n.items) == 0 {
		return
	}

	item := n.items[rand.Intn(len(n.items))]

	n.currentItem = &item
	n.currentLines = item.Content
	n.visibleStart = 0
	n.visibleLines = 3 // 3줄 표시

	n.renderVisibleLines()
}

// -------------------- UTIL --------------------
func (n *NoticeUI) renderVisibleLines() {

	n.linesCont.Objects = nil

	end := n.visibleStart + n.visibleLines
	if end > len(n.currentLines) {
		end = len(n.currentLines)
	}

	for i := n.visibleStart; i < end; i++ {

		line := n.currentLines[i] // ✅ 여기서 선언

		if strings.TrimSpace(line) == "" {
			txt := canvas.NewText(" ", theme.ForegroundColor())
			txt.TextSize = 8
			txt.Alignment = fyne.TextAlignCenter
			n.linesCont.Add(txt)
		} else {
			txt := canvas.NewText(line, theme.ForegroundColor())
			txt.TextSize = 16
			txt.Alignment = fyne.TextAlignCenter
			n.linesCont.Add(txt)
		}
	}

	n.linesCont.Refresh()
}

func (n *NoticeUI) scrollOneLine() {

	if len(n.currentLines) <= n.visibleLines {
		n.pickRandomItem()
		return
	}

	n.visibleStart++

	if n.visibleStart+n.visibleLines > len(n.currentLines) {
		n.pickRandomItem()
		return
	}

	n.renderVisibleLines()
}

func buildTiledBackground(imagePath string, width, height float32) *fyne.Container {

	tileSize := float32(40) // 원하는 타일 크기

	container := container.NewWithoutLayout()

	cols := int(width/tileSize) + 1
	rows := int(height/tileSize) + 1

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {

			tile := canvas.NewImageFromFile(imagePath)
			tile.FillMode = canvas.ImageFillStretch
			tile.Resize(fyne.NewSize(tileSize, tileSize))
			tile.Move(fyne.NewPos(
				float32(x)*tileSize,
				float32(y)*tileSize,
			))

			container.Add(tile)
		}
	}

	container.Resize(fyne.NewSize(width, height))

	return container
}

func randomPattern(dir string) string {
	files, _ := filepath.Glob(filepath.Join(dir, "pt_*.png"))
	if len(files) == 0 {
		return ""
	}
	return files[rand.Intn(len(files))]
}

func openURL(link string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", link)
	case "darwin":
		cmd = exec.Command("open", link)
	default:
		cmd = exec.Command("xdg-open", link)
	}

	_ = cmd.Start()
}
