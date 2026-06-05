// cli/main.go
//
// GoMP3 의 CLI(대화형) 진입점.
// GUI(main.go / ui 패키지)는 전혀 건드리지 않고, service / app / util 패키지를
// 그대로 재사용하여 동일한 단계별 절차(Step1~4)를 콘솔에서 수행한다.
//
// 빌드:
//
//	go build -o dist\gomp3c.exe .\cli   (windowsgui 플래그 없이 → 콘솔 표시)
package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gomp3_gui/app"
	"gomp3_gui/service"
)

var reader = bufio.NewReader(os.Stdin)

func main() {
	fmt.Println("==================================")
	fmt.Println("   GoMP3 CLI (gomp3c)")
	fmt.Println("==================================")

	// =========================
	// AppState 초기화 (GUI와 동일한 진입점 재사용)
	// =========================
	state := app.InitAppState()
	if state == nil {
		fatal("AppState 초기화 실패")
	}

	// =========================
	// 도구 확인 / 자동 설치 (BuildStep1 이 GUI에서 하던 일)
	// =========================
	fmt.Println("· 도구 확인 중 (ffmpeg / yt-dlp) ...")
	ff, err := service.EnsureFFmpeg(state.Paths.Bin)
	if err != nil {
		fatal("ffmpeg 준비 실패: " + err.Error())
	}
	state.Tools.FFmpeg = ff.FFmpeg
	state.Tools.FFplay = ff.FFplay
	state.Tools.FFprobe = ff.FFprobe

	ytdlp, err := service.EnsureYtDlp(state.Paths.Bin)
	if err != nil {
		fatal("yt-dlp 준비 실패: " + err.Error())
	}
	state.Tools.YtDlp = ytdlp

	// =========================
	// Step1 : 메타 수집 + 다운로드
	// =========================
	meta, videoPath := step1(state)

	// =========================
	// 모드 분기 (GUI 의 AutoMode 체크박스에 해당)
	//   - 챕터가 있을 때만 분할 여부를 물어본다 (기본값 Yes).
	//   - 챕터가 없으면 나눌 기준이 없으므로 질문을 생략하고
	//     곧바로 단일 MP3 변환으로 진행한다.
	// =========================
	split := false
	if len(meta.Chapters) > 0 {
		split = askYesNo(
			fmt.Sprintf("\n%d개 챕터가 감지되었습니다. 챕터 기준으로 분할 변환하시겠습니까?", len(meta.Chapters)),
			true,
		)
	}

	if split {
		step4(state, meta, videoPath)
	} else {
		mp3Path := step2(state, meta, videoPath)
		step3(state, meta, mp3Path)
	}

	fmt.Println("\n✅ 모든 작업 완료.")
	fmt.Println("   출력 폴더:", state.Paths.Audio)
}

// =========================================================
// Step1 : 영상 다운로드
// =========================================================
func step1(state *app.AppState) (*service.YTMeta, string) {
	fmt.Println("\n[Step 1] 영상 다운로드")

	var url string
	for {
		url = strings.TrimSpace(prompt("YouTube URL", ""))
		if url != "" {
			break
		}
	}
	state.VideoURL = url

	logFilePath := filepath.Join(state.Paths.Temp, "event.log")
	_ = os.WriteFile(logFilePath, []byte(""), 0644)

	// 1) 메타 수집
	fmt.Println("· 메타 정보 수집 중 ...")
	meta, err := service.FetchYTMeta(state.Tools.YtDlp, url)
	if err != nil {
		fatal("메타 수집 실패: " + err.Error())
	}
	fmt.Printf("  제목   : %s\n", meta.Title)
	fmt.Printf("  업로더 : %s\n", meta.Artist)
	fmt.Printf("  길이   : %s\n", service.FormatTimestamp(meta.Duration))
	if len(meta.Chapters) > 0 {
		fmt.Printf("  챕터   : %d개\n", len(meta.Chapters))
	}

	// 2) 썸네일 다운로드 → 커버 생성용 thumbnail.jpg
	if meta.Thumbnail != "" {
		raw := filepath.Join(state.Paths.Temp, "thumb_raw")
		jpg := filepath.Join(state.Paths.Temp, "thumbnail.jpg")
		if err := service.DownloadFile(meta.Thumbnail, raw); err == nil {
			_ = service.NormalizeThumbnail(raw, jpg)
		}
	}

	// 3) 영상 다운로드
	fmt.Println("· 영상 다운로드 중 ...")
	video, err := service.DownloadVideo(
		state.Tools.YtDlp,
		url,
		state.Paths.Video,
		logFilePath,
		func(line string) { // onLog (이미 [download] 줄만 전달됨)
			fmt.Printf("\r  %-70s", strings.TrimSpace(line))
		},
		nil, // onProgress 미사용 (로그 줄로 충분)
	)
	fmt.Println()
	if err != nil {
		fatal("다운로드 실패: " + err.Error())
	}
	fmt.Println("  완료:", video)

	state.Step1.VideoPath = video
	state.Step1.Meta = meta
	return meta, video
}

// =========================================================
// Step2 : 커버 생성 + 단일 MP3 변환
// =========================================================
func step2(state *app.AppState, meta *service.YTMeta, videoPath string) string {
	fmt.Println("\n[Step 2] MP3 변환")

	artist := prompt("Artist", meta.Artist)
	title := prompt("Title", meta.Title)
	bgMode := askBgMode()

	// 커버 베이스(cover_000.jpg) 생성
	src := filepath.Join(state.Paths.Temp, "thumbnail.jpg")
	cover000 := filepath.Join(state.Paths.Image, "cover_000.jpg")
	fmt.Println("· 커버 이미지 생성 중 ...")
	coverOK := true
	if err := service.ResizeThumbnailToCover(src, cover000, 500, bgMode); err != nil {
		fmt.Println("  ⚠ 커버 생성 실패 (태깅 단계는 건너뜀):", err.Error())
		coverOK = false
	}

	bitrate := askBitrate()

	// MP3 변환 (numbered=false → 기존 YM날짜시분 파일명)
	totalStr := service.FormatTimestamp(meta.Duration)
	logFilePath := filepath.Join(state.Paths.Temp, "event.log")
	fmt.Println("· MP3 변환 중 ...")

	var last time.Time
	mp3Path, err := service.ConvertVideoToMP3(
		state.Tools.FFmpeg, videoPath, state.Paths.Audio, bitrate,
		false, 0, 0, 0, logFilePath,
		func(sec float64) {
			// 너무 잦은 출력 방지: 1초 간격
			if !last.IsZero() && time.Since(last) < time.Second {
				return
			}
			last = time.Now()
			fmt.Printf("\r  변환 중: %s / %s", service.FormatTimestamp(int(sec+0.5)), totalStr)
		},
	)
	fmt.Println()
	if err != nil {
		fatal("MP3 변환 실패: " + err.Error())
	}
	fmt.Println("  완료:", mp3Path)

	state.Step2.Artist = artist
	state.Step2.Title = title
	state.Step2.MP3Path = mp3Path
	state.Step2.Bitrate = bitrate
	state.Step2.CoverGenerated = coverOK
	if coverOK {
		state.Step2.CoverPath = cover000
	}
	return mp3Path
}

// =========================================================
// Step3 : 태그 적용 + (선택) 파일명 변경
// =========================================================
func step3(state *app.AppState, meta *service.YTMeta, mp3Path string) {
	fmt.Println("\n[Step 3] 태그 적용")

	artist := state.Step2.Artist
	title := state.Step2.Title
	confPath := filepath.Join(state.Paths.Conf, "cover.json")

	// 커버에 텍스트 입혀 cover_999.jpg 생성
	coverForTag := ""
	if state.Step2.CoverGenerated && state.Step2.CoverPath != "" {
		cover999 := filepath.Join(state.Paths.Image, "cover_999.jpg")
		if err := service.RenderCoverText(
			state.Step2.CoverPath, cover999, confPath,
			service.CoverTextOption{Artist: artist, Title: title, Size: 500},
		); err != nil {
			fmt.Println("  ⚠ 커버 텍스트 렌더 실패:", err.Error())
		} else {
			coverForTag = cover999
		}
	}

	year := ""
	if len(meta.UploadDate) >= 4 {
		year = meta.UploadDate[:4]
	}

	// 1) 텍스트 태그
	if err := service.ApplyTextTags(mp3Path, artist, title, meta.Album, year, 1, meta.WebpageURL); err != nil {
		fmt.Println("  ⚠ 텍스트 태그 실패:", err.Error())
	} else {
		fmt.Println("  ✓ 텍스트 태그 적용")
	}

	// 2) 커버 이미지
	if coverForTag != "" {
		if err := service.ApplyCoverImage(mp3Path, coverForTag); err != nil {
			fmt.Println("  ⚠ 커버 이미지 적용 실패:", err.Error())
		} else {
			fmt.Println("  ✓ 커버 이미지 적용")
		}
	}

	// 3) 파일명 변경 (선택)
	rule, doRename := askRenameRule()
	if doRename {
		newPath, err := service.RenameMP3(mp3Path, rule, service.ArtistTitleMeta{Artist: artist, Title: title})
		if err != nil {
			fmt.Println("  ⚠ 파일명 변경 실패:", err.Error())
		} else {
			fmt.Println("  ✓ 파일명 변경:", newPath)
			state.Step2.MP3Path = newPath
		}
	}
}

// =========================================================
// Step4 : 분할 변환 (Chapter / Timestamp / Single)
// =========================================================
func step4(state *app.AppState, meta *service.YTMeta, videoPath string) {
	fmt.Println("\n[Step 4] 분할 변환")

	segments := askSegments(meta)
	if len(segments) == 0 {
		fatal("변환할 구간이 없습니다.")
	}

	// 랜덤 커버 후보 (cover_000 / cover_999 제외)
	randCover := pickRandomCover(state.Paths.Image)
	if randCover == "" {
		fatal("사용 가능한 커버 후보(var/image/cover_*.jpg)가 없습니다.")
	}

	confPath := filepath.Join(state.Paths.Conf, "cover.json")
	cover999 := filepath.Join(state.Paths.Image, "cover_999.jpg")
	logFilePath := filepath.Join(state.Paths.Temp, "event.log")
	total := len(segments)

	year := ""
	if len(meta.UploadDate) >= 4 {
		year = meta.UploadDate[:4]
	}

	for i, seg := range segments {
		dur := seg.EndSec - seg.StartSec
		if dur <= 0 {
			fmt.Printf("  ⚠ 잘못된 구간 건너뜀: %s\n", seg.Title)
			continue
		}
		fmt.Printf("\n▶ %d/%d  %s (%s ~ %s)\n", i+1, total, seg.Title,
			service.FormatTimestamp(seg.StartSec), service.FormatTimestamp(seg.EndSec))

		// 1) MP3 변환 (numbered=true → YM날짜N001 ...)
		mp3Path, err := service.ConvertVideoToMP3(
			state.Tools.FFmpeg, videoPath, state.Paths.Audio, "128k",
			true, i+1, seg.StartSec, dur, logFilePath, nil,
		)
		if err != nil {
			fmt.Println("  ⚠ 변환 실패:", err.Error())
			continue
		}
		fmt.Println("  ✓ 변환:", mp3Path)

		// 2) 커버 생성 (cover_999.jpg)
		if err := service.RenderCoverText(randCover, cover999, confPath,
			service.CoverTextOption{Artist: seg.Artist, Title: seg.Title, Size: 500}); err != nil {
			fmt.Println("  ⚠ 커버 생성 실패:", err.Error())
		} else {
			// 3) 태그 적용
			if err := service.ApplyTextTags(mp3Path, seg.Artist, seg.Title, meta.Album, year, i+1, meta.WebpageURL); err != nil {
				fmt.Println("  ⚠ 텍스트 태그 실패:", err.Error())
			}
			if err := service.ApplyCoverImage(mp3Path, cover999); err != nil {
				fmt.Println("  ⚠ 커버 적용 실패:", err.Error())
			}
		}
		fmt.Println("  ✓ 태그 적용 완료")
	}
}

// askSegments: Step4 의 챕터 구간 목록 생성
//
// 분할 여부(챕터 기준)는 이미 main() 의 Y/n 질문에서 결정되었으므로
// 여기서 모드를 다시 묻지 않고 곧바로 챕터 구간을 만든다. (이중 질의 제거)
// CLI 는 간편 변환 수단이므로 timestamp.txt 수동 작성 모드는 제공하지 않는다.
// (사용자가 직접 타임스탬프를 편집하는 흐름은 GUI 에서만 지원)
// 이 함수는 챕터가 있을 때만 호출되지만, 방어적으로 챕터 부재도 처리한다.
func askSegments(meta *service.YTMeta) []service.Segment {
	if len(meta.Chapters) == 0 {
		return singleSegment(meta)
	}

	var segs []service.Segment
	for _, ch := range meta.Chapters {
		segs = append(segs, service.Segment{
			StartSec: ch.StartTime,
			EndSec:   ch.EndTime,
			Title:    ch.Title,
			Artist:   meta.Artist,
		})
	}
	return segs
}

func singleSegment(meta *service.YTMeta) []service.Segment {
	return []service.Segment{{
		StartSec: 0,
		EndSec:   meta.Duration,
		Title:    meta.Title,
		Artist:   meta.Artist,
	}}
}

// pickRandomCover: cover_000 / cover_999 를 제외한 cover_*.jpg 중 무작위 선택
func pickRandomCover(imageDir string) string {
	covers, _ := filepath.Glob(filepath.Join(imageDir, "cover_*.jpg"))
	var candidates []string
	for _, c := range covers {
		base := filepath.Base(c)
		if base == "cover_000.jpg" || base == "cover_999.jpg" {
			continue
		}
		candidates = append(candidates, c)
	}
	if len(candidates) == 0 {
		return ""
	}
	rand.Seed(time.Now().UnixNano())
	return candidates[rand.Intn(len(candidates))]
}

// =========================================================
// 입력 헬퍼
// =========================================================
func prompt(label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

func askYesNo(label string, def bool) bool {
	hint := "y/N"
	if def {
		hint = "Y/n"
	}
	fmt.Printf("%s (%s): ", label, hint)
	line, _ := reader.ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	if line == "" {
		return def
	}
	return line == "y" || line == "yes"
}

func askBgMode() service.BgMode {
	fmt.Println("커버 배경 모드:")
	fmt.Println("  1) 그라데이션(Gradient)  2) 평균색  3) 검은색  4) 배경확장(블러)")
	switch strings.TrimSpace(prompt("선택", "1")) {
	case "2":
		return service.BgAverageColor
	case "3":
		return service.BgBlack
	case "4":
		return service.BgBlurExtend
	default:
		return service.BgAverageColorGradient
	}
}

func askBitrate() string {
	fmt.Println("비트레이트:")
	fmt.Println("  1) 128k  2) 192k  3) 320k")
	switch strings.TrimSpace(prompt("선택", "1")) {
	case "2":
		return "192k"
	case "3":
		return "320k"
	default:
		return "128k"
	}
}

// askRenameRule: 두 번째 반환값이 false 면 이름 변경 안 함
func askRenameRule() (service.FileNameRule, bool) {
	fmt.Println("파일명 규칙:")
	fmt.Println("  1) 변경 안 함 (날짜/시간 유지)  2) Artist - Title  3) Title - Artist")
	switch strings.TrimSpace(prompt("선택", "1")) {
	case "2":
		return service.FileNameRuleArtistTitle, true
	case "3":
		return service.FileNameRuleTitleArtist, true
	default:
		return service.FileNameRuleDate, false
	}
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "\n❌ "+msg)
	os.Exit(1)
}
