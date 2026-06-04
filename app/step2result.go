package app

// Step2Result
// Step2에서 생성된 결과물만 보관
// - 입력(meta)은 Step1Result가 책임진다
type Step2Result struct {

	// 최종 확정 메타데이터
	Artist string
	Title  string

	// 변환된 MP3 파일 경로
	MP3Path string

	// 최종 커버 이미지 경로
	CoverPath      string
	CoverGenerated bool

	// 변환에 사용된 비트레이트
	Bitrate string
}
