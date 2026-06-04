// app/step1_result.go
package app

import "gomp3_gui/service"

type Step1Result struct {
	VideoPath string          // ./video/video.mp4
	Meta      *service.YTMeta // ytmeta.go에서 만든 전체 메타
}
