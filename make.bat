@echo off
setlocal

REM ----------------------------------------
REM 설정
REM ----------------------------------------
REM ffmpeg(fyne) 빌드는 CGo 외부 링커(gcc/ld)를 사용한다.
REM 시스템 MinGW-W64 8.1.0(ld 2.30, 2018)은 go1.26+ 와 비호환이라
REM "이 앱을 PC에서 실행할 수 없습니다"(BAD_EXE_FORMAT) 오류가 발생한다.
REM 작동이 검증된 go1.23.3 으로 고정한다. (MinGW를 최신으로 교체하면 제거 가능)
set GOTOOLCHAIN=go1.23.3
REM 고정 툴체인(go1.23.3) 다운로드/검증에는 GOSUMDB가 필요하다.
REM (전역 GOSUMDB=off 여도 여기서만 일시적으로 켠다 — setlocal 범위)
set GOSUMDB=sum.golang.org
set APP_NAME=gomp3w
set CLI_NAME=gomp3c
set ICON_FILE=gomp3.ico
set MANIFEST_FILE=gomp3w.manifest
set OUTPUT_DIR=dist
set GOOS=windows
set GOARCH=amd64

REM ----------------------------------------
REM 출력 폴더 생성
REM ----------------------------------------
if not exist %OUTPUT_DIR% (
    mkdir %OUTPUT_DIR%
)

REM ----------------------------------------
REM ICO + Manifest -> .syso 변환
REM   - manifest: asInvoker 선언으로 Windows 파일 가상화를 막아
REM               ID3 태그 저장 시 발생하는 UAC 관련 실패를 방지
REM ----------------------------------------
echo Generating Windows resource from %ICON_FILE% + %MANIFEST_FILE% ...
rsrc -ico %ICON_FILE% -manifest %MANIFEST_FILE% -o rsrc.syso
if errorlevel 1 (
    echo Failed to generate rsrc.syso
    exit /b 1
)

REM ----------------------------------------
REM Go 빌드
REM ----------------------------------------
echo Building %APP_NAME%.exe ...
set GOOS=%GOOS%
set GOARCH=%GOARCH%
go build -ldflags="-H=windowsgui" -o %OUTPUT_DIR%\%APP_NAME%.exe
if errorlevel 1 (
    echo Build failed
    exit /b 1
)

REM ----------------------------------------
REM CLI(콘솔) 버전 빌드
REM   - ID3 태그 저장을 위해 동일 manifest를 cli\rsrc.syso 로 적용
REM     (파일 가상화 방지 — GUI와 동일한 이유)
REM   - -H=windowsgui 를 주지 않으므로 콘솔 창이 표시됨
REM ----------------------------------------
echo Generating CLI Windows resource ...
rsrc -ico %ICON_FILE% -manifest %MANIFEST_FILE% -o cli\rsrc.syso
if errorlevel 1 (
    echo Failed to generate cli\rsrc.syso
    exit /b 1
)

echo Building %CLI_NAME%.exe ...
go build -o %OUTPUT_DIR%\%CLI_NAME%.exe .\cli
if errorlevel 1 (
    echo CLI build failed
    exit /b 1
)

REM ----------------------------------------
REM 완료
REM ----------------------------------------
echo Build complete: %OUTPUT_DIR%\%APP_NAME%.exe
echo Build complete: %OUTPUT_DIR%\%CLI_NAME%.exe
pause
