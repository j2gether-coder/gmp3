package service

import (
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

/*
================================================
 Config
================================================
*/

type CoverConfig struct {
	TopBarHeight    int `json:"top_bar_height"`
	BottomBarHeight int `json:"bottom_bar_height"`

	Artist struct {
		X        int `json:"x"`
		Y        int `json:"y"`
		FontSize int `json:"font_size"`
	} `json:"artist"`

	Title struct {
		X        int `json:"x"`
		Y        int `json:"y"`
		FontSize int `json:"font_size"`
	} `json:"title"`

	MaxTextWidth int `json:"max_text_width"`
	LineSpacing  int `json:"line_spacing"`
}

type CoverTextOption struct {
	Artist string
	Title  string
	Size   int // 반드시 500
}

/*
================================================
 Public API (Step3)
================================================
*/

// RenderCoverText
// - 500x500 cover_000.jpg 기준
// - 상단 바 + Artist / Title 렌더링
// - cover_999.jpg 생성
func RenderCoverText(
	srcPath string,
	dstPath string,
	confPath string,
	opt CoverTextOption,
) error {

	cfg, err := loadCoverConfig(confPath)
	if err != nil {
		return err
	}

	srcImg, err := loadImage(srcPath)
	if err != nil {
		return err
	}

	// =========================
	// Size 검증 (Step2 책임 분리)
	// =========================
	if srcImg.Bounds().Dx() != opt.Size || srcImg.Bounds().Dy() != opt.Size {
		return errors.New("source image must be 500x500 (use cover_resize first)")
	}

	dst := image.NewRGBA(srcImg.Bounds())
	draw.Draw(dst, dst.Bounds(), srcImg, image.Point{}, draw.Src)

	useTransparentBars := strings.EqualFold(filepath.Base(srcPath), "cover_000.jpg")

	// =========================
	// 1. 상,하단 바
	// =========================
	if useTransparentBars {
		drawTopBar(dst, cfg.TopBarHeight)
		drawBottomBar(dst, cfg.BottomBarHeight)
	}

	// =========================
	// 2. 텍스트 컬러 (상단바 평균색의 보색)
	// =========================
	bgAvg := averageColor(dst, image.Rect(0, 0, opt.Size, cfg.TopBarHeight))
	textColor := complementaryColor(bgAvg)

	// =========================
	// 3. 폰트 로드 (Windows 기준)
	// =========================
	artistFace := loadFontFace(
		`C:\Windows\Fonts\NanumMyeongjo.ttf`,
		float64(cfg.Artist.FontSize),
	)
	if artistFace == nil {
		artistFace = loadFontFace(`C:\Windows\Fonts\batang.ttc`, float64(cfg.Artist.FontSize))
	}

	titleFace := loadFontFace(
		`C:\Windows\Fonts\NanumGothic.ttf`,
		float64(cfg.Title.FontSize),
	)
	if titleFace == nil {
		titleFace = loadFontFace(`C:\Windows\Fonts\dotum.ttc`, float64(cfg.Title.FontSize))
	}

	if artistFace == nil || titleFace == nil {
		return errors.New("font load failed")
	}

	// =========================
	// 4. 텍스트 렌더링
	// =========================
	drawMultilineText(
		dst,
		cfg.Artist.X,
		cfg.Artist.Y,
		opt.Artist,
		artistFace,
		textColor,
		cfg.MaxTextWidth,
		cfg.LineSpacing,
	)

	drawMultilineText(
		dst,
		cfg.Title.X,
		cfg.Title.Y,
		opt.Title,
		titleFace,
		textColor,
		cfg.MaxTextWidth,
		cfg.LineSpacing,
	)

	// =========================
	// 저장
	// =========================
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return err
	}

	return saveJPEG(dstPath, dst)
}

/*
================================================
 Text Rendering
================================================
*/

func drawMultilineText(
	img *image.RGBA,
	x, y int,
	text string,
	face font.Face,
	col color.Color,
	maxWidth int,
	lineSpacing int,
) {
	lines := wrapText(text, face, maxWidth)

	curY := y
	for _, line := range lines {
		d := &font.Drawer{
			Dst:  img,
			Src:  image.NewUniform(col),
			Face: face,
			Dot:  fixed.P(x, curY),
		}
		d.DrawString(line)
		curY += lineSpacing
	}
}

func wrapText(text string, face font.Face, maxWidth int) []string {
	var lines []string
	var current string

	words := strings.Fields(text)
	d := &font.Drawer{Face: face}

	for _, word := range words {
		test := word
		if current != "" {
			test = current + " " + word
		}

		if d.MeasureString(test).Ceil() > maxWidth {
			lines = append(lines, current)
			current = word
		} else {
			current = test
		}
	}

	if current != "" {
		lines = append(lines, current)
	}

	return lines
}

/*
================================================
 Utilities
================================================
*/

func loadCoverConfig(path string) (*CoverConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg CoverConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func drawTopBar(img *image.RGBA, height int) {
	bar := image.Rect(0, 0, img.Bounds().Dx(), height)
	draw.Draw(
		img,
		bar,
		&image.Uniform{C: color.RGBA{0, 0, 0, 160}},
		image.Point{},
		draw.Over,
	)
}

func drawBottomBar(img *image.RGBA, height int) {
	h := img.Bounds().Dy()
	bar := image.Rect(
		0,
		h-height,
		img.Bounds().Dx(),
		h,
	)

	draw.Draw(
		img,
		bar,
		&image.Uniform{C: color.RGBA{0, 0, 0, 120}}, // 160 -> 100
		image.Point{},
		draw.Over,
	)
}

func loadFontFace(path string, size float64) font.Face {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	ft, err := opentype.Parse(data)
	if err != nil {
		return nil
	}

	face, err := opentype.NewFace(ft, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil
	}

	return face
}

func averageColor(img image.Image, rect image.Rectangle) color.RGBA {
	rect = rect.Intersect(img.Bounds())
	if rect.Empty() {
		return color.RGBA{128, 128, 128, 255}
	}

	var r, g, b uint64
	var count uint64

	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			cr, cg, cb, _ := img.At(x, y).RGBA()
			r += uint64(cr)
			g += uint64(cg)
			b += uint64(cb)
			count++
		}
	}

	return color.RGBA{
		R: uint8((r / count) >> 8),
		G: uint8((g / count) >> 8),
		B: uint8((b / count) >> 8),
		A: 255,
	}
}

func complementaryColor(c color.RGBA) color.RGBA {
	return color.RGBA{
		R: 255 - c.R,
		G: 255 - c.G,
		B: 255 - c.B,
		A: 255,
	}
}
