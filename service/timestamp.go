// service/timestamp.go
package service

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Segment struct {
	StartSec int
	EndSec   int
	Title    string
	Artist   string
}

// --------------------------------------------------
// HH:MM:SS → seconds
// --------------------------------------------------
func parseTimestampToSec(ts string) int {
	parts := strings.Split(ts, ":")
	if len(parts) == 2 {
		// MM:SS
		m, _ := strconv.Atoi(parts[0])
		s, _ := strconv.Atoi(parts[1])
		return m*60 + s
	} else if len(parts) == 3 {
		h, _ := strconv.Atoi(parts[0])
		m, _ := strconv.Atoi(parts[1])
		s, _ := strconv.Atoi(parts[2])
		return h*3600 + m*60 + s
	}
	return -1
}

// --------------------------------------------------
// seconds → HH:MM:SS
// --------------------------------------------------
func FormatTimestamp(sec int) string {
	if sec < 0 {
		sec = 0
	}

	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60

	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// --------------------------------------------------
// 사용자 timestamp 파싱 (Step4 전용)
// --------------------------------------------------
func ParseUserTimestampFile(
	path string,
	totalDuration int,
	separator string,
	fallbackArtist string,
	inputArtistFirst bool,
) ([]Segment, error) {

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	type tempItem struct {
		StartSec int
		Title    string
		Artist   string
	}

	var temp []tempItem

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {

		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		start := parseTimestampToSec(parts[0])
		if start < 0 {
			continue
		}

		raw := strings.TrimSpace(parts[1])

		title := raw
		artist := fallbackArtist

		if separator != "" && strings.Contains(raw, separator) {
			split := strings.SplitN(raw, separator, 2)
			if len(split) < 2 {
				title = raw
				artist = fallbackArtist
			} else {
				left := strings.TrimSpace(split[0])
				right := strings.TrimSpace(split[1])
				if inputArtistFirst {
					artist = left
					title = right
				} else {
					title = left
					artist = right
				}
			}
		}

		temp = append(temp, tempItem{
			StartSec: start,
			Title:    title,
			Artist:   artist,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(temp) == 0 {
		return nil, fmt.Errorf("no valid timestamp entries found")
	}

	var segments []Segment

	for i := 0; i < len(temp); i++ {

		start := temp[i].StartSec
		end := totalDuration

		if i < len(temp)-1 {
			end = temp[i+1].StartSec
		}

		if end <= start {
			continue
		}

		segments = append(segments, Segment{
			StartSec: start,
			EndSec:   end,
			Title:    temp[i].Title,
			Artist:   temp[i].Artist,
		})
	}

	return segments, nil
}
