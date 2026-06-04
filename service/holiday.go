package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gomp3_gui/util"
)

var (
	loadOnce sync.Once
	loadErr  error
	data     holidayFile
)

type holidayFile struct {
	Version  string         `json:"version"`
	Country  string         `json:"country"`
	Timezone string         `json:"timezone"`
	Year     int            `json:"year"`
	Events   []holidayEvent `json:"events"`
}

type holidayEvent struct {
	Date string `json:"date"` // MM-DD
	Name string `json:"name"`
	Type string `json:"type"`
}

var priority = map[string]int{
	"national":   1,
	"substitute": 2,
}

func BuildTodayNotice(now time.Time) (string, error) {
	if err := loadData(); err != nil {
		return "", err
	}

	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		return "", err
	}

	now = now.In(loc)

	today := now.Format("01-02")

	var todays []holidayEvent

	for _, e := range data.Events {

		if e.Date == today {
			todays = append(todays, e)
		}
	}

	if len(todays) == 0 {
		return "", nil
	}

	sort.Slice(todays, func(i, j int) bool {
		return priority[todays[i].Type] < priority[todays[j].Type]
	})

	return buildSentence(todays), nil
}

func loadData() error {
	loadOnce.Do(func() {

		paths, err := util.GetAppPaths()
		if err != nil {
			loadErr = err
			return
		}

		path := filepath.Join(paths.Data, "holidays.json")

		file, err := os.Open(path)
		if err != nil {
			loadErr = err
			return
		}
		defer file.Close()

		if err := json.NewDecoder(file).Decode(&data); err != nil {
			loadErr = err
			return
		}
	})

	return loadErr
}

func buildSentence(events []holidayEvent) string {
	names := make([]string, len(events))

	for i, e := range events {
		names[i] = e.Name
	}

	if len(names) == 1 {
		return fmt.Sprintf("오늘은 %s입니다.", names[0])
	}

	if len(names) == 2 {
		return fmt.Sprintf("오늘은 %s이고 %s입니다.", names[0], names[1])
	}

	var b strings.Builder

	b.WriteString("오늘은 ")

	for i, name := range names {

		switch {

		case i == 0:
			b.WriteString(name)

		case i == len(names)-1:
			b.WriteString("이며 ")
			b.WriteString(name)

		default:
			b.WriteString("이고 ")
			b.WriteString(name)
		}
	}

	b.WriteString("입니다.")

	return b.String()
}

const holidayAPIKey = "V1YHKpaR6s5NbJWm91kXNVMN6bTyAZQghsdkQk9gJNZLKTMH1%2FxfATuDHb7IfGDGsY%2BMlmH%2FnsYQwiQBnJ%2FVDw%3D%3D"

func EnsureHolidayYear(year int) error {
	events, err := fetchPublicHolidays(year)
	if err != nil {
		return err
	}

	data = holidayFile{
		Version:  "1.0.0",
		Country:  "KR",
		Timezone: "Asia/Seoul",
		Year:     year,
		Events:   events,
	}

	return saveHolidayFile()
}

func fetchPublicHolidays(year int) ([]holidayEvent, error) {
	url := fmt.Sprintf(
		"https://apis.data.go.kr/B090041/openapi/service/SpcdeInfoService/getRestDeInfo?serviceKey=%s&solYear=%d&_type=json",
		holidayAPIKey,
		year,
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	type item struct {
		DateName string `json:"dateName"`
		LocDate  int    `json:"locdate"`
	}

	var result struct {
		Response struct {
			Body struct {
				Items struct {
					Item []item `json:"item"`
				} `json:"items"`
			} `json:"body"`
		} `json:"response"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var events []holidayEvent

	for _, it := range result.Response.Body.Items.Item {

		date := fmt.Sprintf("%08d", it.LocDate)

		mmdd := date[4:6] + "-" + date[6:8]

		name := strings.TrimSpace(it.DateName)

		typ := "national"

		if strings.Contains(name, "대체") {
			typ = "substitute"
		}

		events = append(events, holidayEvent{
			Date: mmdd,
			Name: name,
			Type: typ,
		})
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Date < events[j].Date
	})

	return events, nil
}

func saveHolidayFile() error {
	paths, err := util.GetAppPaths()
	if err != nil {
		return err
	}

	path := filepath.Join(paths.Data, "holidays.json")

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, b, 0644)
}
