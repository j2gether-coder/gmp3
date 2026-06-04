package service

import (
	"fmt"
	"os"
	"strconv"

	id3 "github.com/bogem/id3v2/v2"
)

// ApplyMP3Tags
// 텍스트 태그와 커버 이미지를 순차적으로 분리해서 적용한다.
//   1) ApplyTextTags  : 텍스트 프레임만 저장 (보통 padding 안에 들어가서 안전)
//   2) ApplyCoverImage: 커버 이미지 프레임만 저장 (MP3 전체 재작성이 일어남)
// 1) 단계까지 성공하면 커버 적용이 실패해도 텍스트 태그는 보존된다.
func ApplyMP3Tags(
	mp3Path string,
	artist string,
	title string,
	album string,
	year string,
	track int,
	coverPath string,
	sourceURL string,
) error {

	if err := ApplyTextTags(mp3Path, artist, title, album, year, track, sourceURL); err != nil {
		return err
	}

	if coverPath == "" {
		return nil
	}

	return ApplyCoverImage(mp3Path, coverPath)
}

// ApplyTextTags
// MP3 파일에 텍스트 ID3 태그만 적용한다 (커버 이미지 제외).
func ApplyTextTags(
	mp3Path string,
	artist string,
	title string,
	album string,
	year string,
	track int,
	sourceURL string,
) error {

	tag, err := id3.Open(mp3Path, id3.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("id3 open 실패(text): %w", err)
	}
	defer tag.Close()

	tag.SetArtist(artist)
	tag.SetTitle(title)

	if album != "" {
		tag.SetAlbum(album)
	}

	if year != "" {
		tag.SetYear(year)
	}

	tag.AddTextFrame(
		tag.CommonID("Track number/Position in set"),
		id3.EncodingUTF8,
		strconv.Itoa(track),
	)

	tag.AddCommentFrame(
		id3.CommentFrame{
			Encoding:    id3.EncodingUTF8,
			Language:    "eng",
			Description: "",
			Text:        "[출처] " + sourceURL,
		},
	)

	if err := tag.Save(); err != nil {
		return fmt.Errorf("id3 save 실패(text): %w", err)
	}
	return nil
}

// ApplyCoverImage
// MP3 파일에 커버 이미지(Attached picture) 프레임만 적용한다.
// 텍스트 태그 적용 이후 별도 호출하여 부분 성공이 가능하도록 분리되어 있다.
func ApplyCoverImage(mp3Path string, coverPath string) error {

	img, err := os.ReadFile(coverPath)
	if err != nil {
		return fmt.Errorf("cover read 실패: %w", err)
	}

	tag, err := id3.Open(mp3Path, id3.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("id3 open 실패(cover): %w", err)
	}
	defer tag.Close()

	tag.DeleteFrames(tag.CommonID("Attached picture"))

	tag.AddAttachedPicture(id3.PictureFrame{
		Encoding:    id3.EncodingUTF8,
		MimeType:    "image/jpeg",
		PictureType: id3.PTFrontCover,
		Description: "Cover",
		Picture:     img,
	})

	if err := tag.Save(); err != nil {
		return fmt.Errorf("id3 save 실패(cover): %w", err)
	}
	return nil
}
