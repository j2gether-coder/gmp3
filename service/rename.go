package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileNameRule string

const (
	FileNameRuleDate        FileNameRule = "기본 (날짜/시간)"
	FileNameRuleArtistTitle FileNameRule = "Artist - Title"
	FileNameRuleTitleArtist FileNameRule = "Title - Artist"
)

// RenameMP3 renames mp3 file by rule
func RenameMP3(
	mp3Path string,
	rule FileNameRule,
	meta ArtistTitleMeta,
) (string, error) {

	dir := filepath.Dir(mp3Path)
	ext := filepath.Ext(mp3Path)

	var baseName string

	switch rule {

	case FileNameRuleArtistTitle:
		if meta.Artist == "" || meta.Title == "" {
			return "", fmt.Errorf("artist 또는 title 이 비어 있습니다")
		}
		baseName = fmt.Sprintf("%s - %s", meta.Artist, meta.Title)

	case FileNameRuleTitleArtist:
		if meta.Artist == "" || meta.Title == "" {
			return "", fmt.Errorf("artist 또는 title 이 비어 있습니다")
		}
		baseName = fmt.Sprintf("%s - %s", meta.Title, meta.Artist)

	default:
		now := time.Now()
		baseName = fmt.Sprintf(
			"YM%02d%02d%02dT%02d%02d",
			now.Year()%100,
			now.Month(),
			now.Day(),
			now.Hour(),
			now.Minute(),
		)
	}

	baseName = sanitizeFileName(baseName)

	newPath := uniqueFilePath(dir, baseName, ext)

	if err := os.Rename(mp3Path, newPath); err != nil {
		return "", err
	}

	return newPath, nil
}

func sanitizeFileName(s string) string {
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, c := range invalid {
		s = strings.ReplaceAll(s, c, "_")
	}
	return strings.TrimSpace(s)
}

func uniqueFilePath(dir, base, ext string) string {
	path := filepath.Join(dir, base+ext)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}

	for i := 1; ; i++ {
		name := fmt.Sprintf("%s (%d)%s", base, i, ext)
		path = filepath.Join(dir, name)

		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path
		}
	}
}
