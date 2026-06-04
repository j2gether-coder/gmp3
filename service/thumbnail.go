package service

import (
	"image"
	"image/jpeg"
	"os"

	_ "image/png"

	_ "image/gif"

	_ "golang.org/x/image/webp"
)

func NormalizeThumbnail(srcPath, dstJPG string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return err
	}

	out, err := os.Create(dstJPG)
	if err != nil {
		return err
	}
	defer out.Close()

	return jpeg.Encode(out, img, &jpeg.Options{Quality: 90})
}
