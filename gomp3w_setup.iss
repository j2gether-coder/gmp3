#define MyAppName "GoMP3w"
#define MyAppVersion "1.0.3"
#define MyAppPublisher "com.gomp3w.downloader"
#define MyAppExeName "GoMP3w.exe"

[Setup]
AppId={{F3C1C1A2-4F4D-4A1C-B1E2-123456789001}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
DefaultDirName={sd}\{#MyAppName}
DefaultGroupName={#MyAppName}
OutputDir=C:\Users\COADMIN\Documents\gomp3_setup\dist
OutputBaseFilename=GoMP3_Setup
SetupIconFile=C:\Users\COADMIN\Documents\gomp3_setup\bin\gomp3.ico
Compression=lzma
SolidCompression=yes
PrivilegesRequired=lowest
DisableDirPage=no

[Languages]
Name: "korean"; MessagesFile: "compiler:Languages\Korean.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
Source: "C:\Users\COADMIN\Documents\gomp3_setup\bin\GoMP3w.exe"; DestDir: "{app}\bin"; Flags: ignoreversion
Source: "C:\Users\COADMIN\Documents\gomp3_setup\bin\GoMP3c.exe"; DestDir: "{app}\bin"; Flags: ignoreversion
Source: "C:\Users\COADMIN\Documents\gomp3_setup\bin\GoMP3.ico"; DestDir: "{app}\bin"; Flags: ignoreversion
Source: "C:\Users\COADMIN\Documents\gomp3_setup\var\*"; DestDir: "{app}\var"; Flags: ignoreversion recursesubdirs createallsubdirs

[Dirs]
Name: "{app}\audio"
Name: "{app}\video"
Name: "{app}\bin"
Name: "{app}\var"
Name: "{app}\var\conf"
Name: "{app}\var\assets"
Name: "{app}\var\data"
Name: "{app}\var\image"
Name: "{app}\var\temp"
Name: "{app}\var\work"

[Icons]
; 바로가기 생성 (작업 디렉토리를 루트인 {app}으로 설정하여 경로 호환성 확보)
Name: "{group}\{#MyAppName}"; Filename: "{app}\bin\{#MyAppExeName}"; WorkingDir: "{app}"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\bin\{#MyAppExeName}"; WorkingDir: "{app}"; Tasks: desktopicon

[Run]
; 설치 완료 후 실행 옵션
Filename: "{app}\bin\{#MyAppExeName}"; Description: "{cm:LaunchProgram,{#StringChange(MyAppName, '&', '&&')}}"; Flags: nowait postinstall skipifsilent

[UninstallDelete]
; bin 전체 제거
Type: filesandordirs; Name: "{app}\bin"
