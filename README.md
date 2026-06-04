# GoMP3w

GoMP3w는 YouTube 영상에서 오디오를 추출해 MP3로 변환하고, 커버 이미지와 ID3 태그를 적용하는 Windows용 GUI 프로그램입니다. Fyne 기반 GUI와 콘솔용 CLI를 함께 제공합니다.

## 주요 기능

- YouTube URL 메타데이터 조회 및 영상 다운로드
- FFmpeg를 이용한 MP3 변환
- 썸네일 기반 500x500 커버 이미지 생성
- Artist, Title, Album, Year, Track, 출처 URL ID3 태그 적용
- MP3 미리 듣기 및 원본 영상 미리 보기
- 파일명 변경 규칙 지원
- 챕터, 사용자 timestamp, 전체 영상 기준 분할 변환
- `ffmpeg`, `ffplay`, `ffprobe`, `yt-dlp` 자동 준비

## 실행 환경

- Windows 64-bit
- Go 1.23.3
- 인터넷 연결

프로그램 실행 중 필요한 도구가 `bin` 폴더에 없으면 자동으로 다운로드합니다.

- FFmpeg: `https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip`
- yt-dlp: `https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe`

## 빠른 실행

개발 환경에서 GUI를 바로 실행합니다.

```bat
go run .
```

콘솔 버전은 다음 명령으로 실행할 수 있습니다.

```bat
go run .\cli
```

## 빌드

Windows 실행 파일은 `make.bat`로 생성합니다.

```bat
make.bat
```

빌드 결과는 `dist` 폴더에 생성됩니다.

- `dist\gomp3w.exe`: GUI 버전
- `dist\gomp3c.exe`: CLI 버전

`make.bat`는 아이콘과 manifest를 `.syso` 리소스로 만들기 위해 `rsrc` 명령을 사용합니다. 설치되어 있지 않다면 다음 명령으로 설치합니다.

```bat
go install github.com/akavel/rsrc@latest
```

## 사용 흐름

### GUI

1. YouTube URL을 입력하고 `Download`를 누릅니다.
2. 메타데이터, 썸네일, 영상을 내려받습니다.
3. 일반 변환 모드에서는 Artist/Title을 확인하고 커버 이미지를 생성합니다.
4. 비트레이트를 선택한 뒤 MP3로 변환합니다.
5. 커버와 태그를 저장하고 필요하면 파일명을 변경합니다.

첫 화면의 체크박스를 선택하면 분할 변환 화면으로 바로 이동합니다.

### 분할 변환

분할 변환은 세 가지 모드를 지원합니다.

- `Chapter`: 영상에 포함된 챕터 정보를 기준으로 분할
- `Timestamp`: `var\temp\timestamp.txt`에 작성한 시간표 기준으로 분할
- `Single`: 전체 영상을 하나의 MP3로 변환

Timestamp 파일 형식 예시는 다음과 같습니다.

```txt
00:00:00 Intro - ArtistName
00:03:12 SongTitle - ArtistName
00:07:45 Another Song - ArtistName
```

구분자는 UI에서 지정할 수 있으며, `Artist - Title` 입력 형식도 선택할 수 있습니다.

### CLI

CLI 버전은 GUI와 같은 처리 흐름을 콘솔에서 순서대로 진행합니다.

```bat
go run .\cli
```

실행 후 URL, 분할 여부, Artist/Title, 커버 배경 모드, 비트레이트, 파일명 규칙 등을 프롬프트에 맞춰 입력합니다.

## 폴더 구조

```txt
.
+-- app\        앱 상태 및 단계별 결과 모델
+-- cli\        콘솔 진입점
+-- service\    다운로드, 변환, 태그, 커버, 타임스탬프 처리
+-- ui\         Fyne GUI 화면
+-- util\       경로, 언어 리소스 유틸리티
+-- var\
|   +-- assets\  기본 이미지
|   +-- conf\    커버/재생 설정
|   +-- data\    다국어 문구, 공지, 휴일 데이터
|   +-- image\   생성된 커버 이미지
|   +-- temp\    임시 파일과 로그
+-- audio\      변환된 MP3 출력
+-- video\      다운로드된 영상
+-- bin\        ffmpeg, ffplay, ffprobe, yt-dlp
+-- dist\       빌드 결과물
```

## 출력 파일

일반 변환 파일명은 날짜와 시간을 기준으로 생성됩니다.

```txt
audio\YM260604T1430.mp3
```

분할 변환 파일명은 순번을 포함합니다.

```txt
audio\YM260604N001.mp3
audio\YM260604N002.mp3
```

태그 저장 후에는 다음 파일명 규칙 중 하나를 선택할 수 있습니다.

- 변경 없음
- `Artist - Title`
- `Title - Artist`

## 설정 파일

- `var\conf\cover.json`: 커버 이미지 텍스트 위치, 글자 크기, 줄 간격 설정
- `var\conf\play_video.json`: FFplay 미리 보기 창 크기와 동작 설정

## 개발 메모

- GUI 진입점은 `main.go`입니다.
- CLI 진입점은 `cli\main.go`입니다.
- 공통 비즈니스 로직은 `service` 패키지에 모여 있습니다.
- 실행 시 작업 폴더 기준으로 `audio`, `video`, `var\temp`, `var\image` 등이 자동 생성됩니다.
- 배포 형태에서 실행 파일이 `bin` 폴더 안에 있으면 상위 폴더를 앱 루트로 인식합니다.

## 주의사항

- YouTube 및 각 콘텐츠의 이용 약관과 저작권을 준수해 사용하세요.
- 긴 영상이나 많은 구간을 분할 변환할 때는 처리 시간이 오래 걸릴 수 있습니다.
- 백신 또는 UAC 정책에 따라 MP3 태그 저장이 실패할 수 있습니다. 이 경우 텍스트 태그와 커버 적용은 단계별로 처리되어 일부 결과가 남을 수 있습니다.
