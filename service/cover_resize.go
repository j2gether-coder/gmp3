package service

import (
	"errors"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"

	xdraw "golang.org/x/image/draw"
)

/*
================================================
 Step2 : Thumbnail → 500x500 Cover Base
================================================
*/

// CoverBackgroundMode
// Step2 UI 라디오 버튼과 1:1 대응
type BgMode int

const (
	BgBlack BgMode = iota
	BgAverageColor
	BgAverageColorGradient
	BgBlurExtend // 원본 늘림 + 블러
)

// ResizeThumbnailToCover
// - Step1 썸네일 → Step3이 신뢰하는 cover_000.jpg 생성
func ResizeThumbnailToCover(
	srcPath string,
	dstPath string,
	size int,
	mode BgMode,
) error {

	if size <= 0 {
		return errors.New("invalid size")
	}

	srcImg, err := loadImage(srcPath)
	if err != nil {
		return err
	}

	// 전면 이미지 (Aspect 유지, 내부 맞춤)
	fg := resizeFit(srcImg, size)

	// 배경 선택
	var bg *image.RGBA
	switch mode {
	case BgBlack:
		bg = backgroundBlack(size)
	case BgAverageColor:
		bg = backgroundAverageColor(srcImg, size)
	case BgAverageColorGradient:
		bg = backgroundAverageGradient(srcImg, size)
	case BgBlurExtend:
		bg = backgroundBlurExtend(srcImg, size)
	default:
		return errors.New("unknown background mode")
	}

	// 합성
	out := composeCenter(bg, fg)

	// 저장
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return err
	}

	return saveJPEG(dstPath, out)
}

/*
================================================
 Foreground (공통)
================================================
*/

// resizeFit
// - Aspect Ratio 유지
// - size 안에 맞춤 (여백 발생 가능)
func resizeFit(src image.Image, size int) *image.RGBA {
	sb := src.Bounds()
	srcW := sb.Dx()
	srcH := sb.Dy()

	scaleX := float64(size) / float64(srcW)
	scaleY := float64(size) / float64(srcH)

	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	w := int(float64(srcW) * scale)
	h := int(float64(srcH) * scale)

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, sb, xdraw.Over, nil)

	return dst
}

/*
================================================
 Background Variants
================================================
*/

func backgroundBlack(size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(
		img,
		img.Bounds(),
		&image.Uniform{C: color.RGBA{0, 0, 0, 255}},
		image.Point{},
		draw.Src,
	)
	return img
}

func backgroundAverageColor(src image.Image, size int) *image.RGBA {
	avg := averageColor(src, src.Bounds())
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(
		img,
		img.Bounds(),
		&image.Uniform{C: avg},
		image.Point{},
		draw.Src,
	)
	return img
}

func adjustBrightness(c color.RGBA, factor float64) color.RGBA {
	clamp := func(v int) uint8 {
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return uint8(v)
	}

	return color.RGBA{
		R: clamp(int(float64(c.R) * factor)),
		G: clamp(int(float64(c.G) * factor)),
		B: clamp(int(float64(c.B) * factor)),
		A: 255,
	}
}

func backgroundAverageGradient(src image.Image, size int) *image.RGBA {
	avg := averageColor(src, src.Bounds())

	top := adjustBrightness(avg, 0.65)    // 더 어둡게
	bottom := adjustBrightness(avg, 1.15) // 더 밝게

	img := image.NewRGBA(image.Rect(0, 0, size, size))

	for y := 0; y < size; y++ {
		t := float64(y) / float64(size-1)

		r := uint8(float64(top.R)*(1-t) + float64(bottom.R)*t)
		g := uint8(float64(top.G)*(1-t) + float64(bottom.G)*t)
		b := uint8(float64(top.B)*(1-t) + float64(bottom.B)*t)

		rowColor := color.RGBA{r, g, b, 255}

		draw.Draw(
			img,
			image.Rect(0, y, size, y+1),
			&image.Uniform{C: rowColor},
			image.Point{},
			draw.Src,
		)
	}

	return img
}

// backgroundBlurExtend
// - 원본을 꽉 채우게 resize
// - 블러 적용
func backgroundBlurExtend(src image.Image, size int) *image.RGBA {
	bg := resizeFill(src, size)
	blurBox(bg, 8)
	return bg
}

/*
================================================
 Background Helpers
================================================
*/

// resizeFill
// - Aspect 유지
// - size를 꽉 채움 (crop 발생)
func resizeFill(src image.Image, size int) *image.RGBA {
	sb := src.Bounds()

	scaleX := float64(size) / float64(sb.Dx())
	scaleY := float64(size) / float64(sb.Dy())

	scale := scaleX
	if scaleY > scaleX {
		scale = scaleY
	}

	w := int(float64(sb.Dx()) * scale)
	h := int(float64(sb.Dy()) * scale)

	tmp := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(tmp, tmp.Bounds(), src, sb, xdraw.Over, nil)

	x0 := (w - size) / 2
	y0 := (h - size) / 2

	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(dst, dst.Bounds(), tmp, image.Point{x0, y0}, draw.Src)

	return dst
}

// blurBox
// 단순 Box Blur (가볍고 충분히 예쁨)
func blurBox(img *image.RGBA, radius int) {
	b := img.Bounds()
	tmp := image.NewRGBA(b)

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			var r, g, b2, a uint32
			var count uint32

			for dy := -radius; dy <= radius; dy++ {
				for dx := -radius; dx <= radius; dx++ {
					px := x + dx
					py := y + dy
					if px >= b.Min.X && px < b.Max.X && py >= b.Min.Y && py < b.Max.Y {
						cr, cg, cb, ca := img.At(px, py).RGBA()
						r += cr
						g += cg
						b2 += cb
						a += ca
						count++
					}
				}
			}

			tmp.SetRGBA(x, y, color.RGBA{
				R: uint8((r / count) >> 8),
				G: uint8((g / count) >> 8),
				B: uint8((b2 / count) >> 8),
				A: uint8((a / count) >> 8),
			})
		}
	}

	draw.Draw(img, b, tmp, image.Point{}, draw.Src)
}

/*
================================================
 Compose
================================================
*/

func composeCenter(bg *image.RGBA, fg *image.RGBA) *image.RGBA {
	bw := bg.Bounds().Dx()
	fw := fg.Bounds().Dx()
	fh := fg.Bounds().Dy()

	ox := (bw - fw) / 2
	oy := (bw - fh) / 2

	draw.Draw(
		bg,
		image.Rect(ox, oy, ox+fw, oy+fh),
		fg,
		image.Point{},
		draw.Over,
	)

	return bg
}
