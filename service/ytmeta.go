package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os/exec"
	"syscall"
)

// ========================
// Public Model
// ========================

// YTMeta
// yt-dlp --dump-json 전체 메타 보관
type YTMeta struct {
	// UI 노출
	Title     string
	Artist    string
	Thumbnail string

	// Tag / 확장용
	Album       string
	UploadDate  string
	Duration    int
	Channel     string
	Description string
	WebpageURL  string

	// Step4 MP3 분할용
	Chapters []YTChapter

	// Raw JSON (확장 대비)
	Raw map[string]interface{}
}

type YTChapter struct {
	StartTime int
	EndTime   int
	Title     string
}

// ========================
// Public API
// ========================
// FetchYTMeta
func FetchYTMeta(ytdlpPath, url string) (*YTMeta, error) {

	if ytdlpPath == "" {
		return nil, errors.New("yt-dlp path not set")
	}
	if url == "" {
		return nil, errors.New("URL is empty")
	}

	cmd := exec.Command(
		ytdlpPath,
		"--no-playlist",
		"--dump-json",
		url,
	)
	// Windows GUI 프로그램 대응
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp executed failed: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("JSON parsing failed: %w", err)
	}

	meta := &YTMeta{
		Raw: raw,
	}

	// 라이브 영상 차단
	if v, ok := raw["is_live"].(bool); ok && v {
		return nil, errors.New("Live videos are not supported.")
	}

	// ========================
	// 안전한 필드 추출
	// ========================
	meta.Title = getString(raw, "title")
	meta.Artist = getString(raw, "uploader")
	meta.Thumbnail = getString(raw, "thumbnail")

	meta.Channel = getString(raw, "channel")
	meta.Description = getString(raw, "description")
	meta.UploadDate = getString(raw, "upload_date")
	meta.WebpageURL = getString(raw, "webpage_url")

	if v, ok := raw["duration"].(float64); ok {
		meta.Duration = int(math.Round(v)) //int(v + 0.5) //반올림
	}

	// duration 검증
	if meta.Duration <= 0 {
		return nil, errors.New("Cannot retrieve video duration.")
	}

	// ========================
	// Chapters 추출
	// ========================
	if chRaw, ok := raw["chapters"].([]interface{}); ok {
		for _, c := range chRaw {
			if chMap, ok := c.(map[string]interface{}); ok {

				var ch YTChapter

				if v, ok := chMap["start_time"].(float64); ok {
					ch.StartTime = int(v)
				}

				if v, ok := chMap["end_time"].(float64); ok {
					ch.EndTime = int(v)
				}

				ch.Title = getString(chMap, "title")

				// 안전 장치
				if ch.EndTime > ch.StartTime {
					meta.Chapters = append(meta.Chapters, ch)
				}
			}
		}
	}

	// 앨범 이름 전략
	// 1️⃣ album 필드
	// 2️⃣ playlist
	// 3️⃣ channel fallback
	meta.Album = getString(raw, "album")
	if meta.Album == "" {
		meta.Album = getString(raw, "playlist")
	}
	if meta.Album == "" {
		meta.Album = meta.Artist
	}

	return meta, nil
}

// ========================
// Helpers
// ========================
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
