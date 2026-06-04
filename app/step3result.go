package app

// Step3Result
// Step3(태그 & 파일 정리) 결과 요약
type Step3Result struct {

	// 태그 적용 여부
	TagApplied bool

	// 선택된 파일명 규칙
	// - 기본 (날짜/시간)
	// - Artist - Title
	// - Title - Artist
	//Meta         ArtistTitleMeta
	FileNameRule string

	// 최종 MP3 파일 경로
	FinalPath string
}
