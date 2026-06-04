package ui

import (
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type progressMode int

const (
	ProgressIdle progressMode = iota
	ProgressDownload
	ProgressPlay
)

type Progress struct {
	bar   fyne.CanvasObject // ProgressBar 또는 ProgressBarInfinite
	label *widget.Label

	mu       sync.Mutex
	mode     progressMode
	start    time.Time
	duration float64
}

// NewProgress
// bar: fyne.CanvasObject (ProgressBar 또는 ProgressBarInfinite)
// label: Label로 상태/시간 표시
func NewProgress(bar fyne.CanvasObject, label *widget.Label) *Progress {
	return &Progress{
		bar:   bar,
		label: label,
		mode:  ProgressIdle,
	}
}

// Reset
func (p *Progress) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.mode = ProgressIdle
	fyne.Do(func() {
		switch b := p.bar.(type) {
		case *widget.ProgressBar:
			b.SetValue(0)
			b.Hide()
		case *widget.ProgressBarInfinite:
			b.Stop()
			b.Hide()
		}
		p.label.SetText("")
	})
}

// =======================
// 다운로드
// =======================
func (p *Progress) StartDownload() {
	p.mu.Lock()
	p.mode = ProgressDownload
	p.mu.Unlock()

	fyne.Do(func() {
		switch b := p.bar.(type) {
		case *widget.ProgressBarInfinite:
			b.Start()
			b.Show()
		case *widget.ProgressBar:
			b.SetValue(0)
			b.Show()
		}
		p.label.SetText("Downloading...")
	})
}

// UpdateDownload: 외부에서 퍼센트 주입 가능
// percent: 0.0 ~ 1.0
func (p *Progress) UpdateDownload(percent float64, msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.mode != ProgressDownload {
		return
	}

	if percent < 0 {
		percent = 0
	} else if percent > 1 {
		percent = 1
	}

	fyne.Do(func() {
		if bar, ok := p.bar.(*widget.ProgressBar); ok {
			bar.SetValue(percent)
		}
		if msg != "" {
			p.label.SetText(msg)
		} else {
			p.label.SetText(fmt.Sprintf("%.0f%%", percent*100))
		}
	})
}

func (p *Progress) FinishDownload() {
	p.mu.Lock()
	p.mode = ProgressIdle
	p.mu.Unlock()

	fyne.Do(func() {
		switch b := p.bar.(type) {
		case *widget.ProgressBarInfinite:
			b.Stop()
			b.Hide()
		case *widget.ProgressBar:
			b.SetValue(1)
		}
		p.label.SetText("Donwload completed.")
	})
}

func (p *Progress) FailDownload(msg string) {
	p.mu.Lock()
	p.mode = ProgressIdle
	p.mu.Unlock()

	fyne.Do(func() {
		switch b := p.bar.(type) {
		case *widget.ProgressBarInfinite:
			b.Stop()
			b.Hide()
		case *widget.ProgressBar:
			b.SetValue(0)
		}
		p.label.SetText(msg)
	})
}

// =======================
// 재생
// =======================
func (p *Progress) StartPlay(durationSec float64) {
	if durationSec <= 0 {
		return
	}

	p.mu.Lock()
	p.mode = ProgressPlay
	p.start = time.Now()
	p.duration = durationSec
	p.mu.Unlock()

	fyne.Do(func() {
		switch b := p.bar.(type) {
		case *widget.ProgressBar:
			b.SetValue(0)
			b.Show()
		case *widget.ProgressBarInfinite:
			b.Hide() // 재생 시에는 무한바 사용 안함
		}
		p.label.SetText("재생 시작")
	})
}

func (p *Progress) UpdatePlay(currentSec float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.mode != ProgressPlay || p.duration <= 0 {
		return
	}

	if currentSec > p.duration {
		currentSec = p.duration
	}

	percent := currentSec / p.duration

	fyne.Do(func() {
		// ProgressBar는 %로 표시
		if bar, ok := p.bar.(*widget.ProgressBar); ok {
			bar.SetValue(percent)
		}
		// Label은 timestamp로 표시
		p.label.SetText(fmt.Sprintf(
			"%s / %s",
			formatTime(currentSec),
			formatTime(p.duration),
		))
	})
}

func (p *Progress) FinishPlay() {
	p.mu.Lock()
	p.mode = ProgressIdle
	p.mu.Unlock()

	fyne.Do(func() {
		if bar, ok := p.bar.(*widget.ProgressBar); ok {
			bar.SetValue(1)
		}
		p.label.SetText("Play completed")
	})
}

// =======================
// time formatting
// =======================
func formatTime(sec float64) string {
	total := int(sec + 0.5)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}
