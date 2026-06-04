package util

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type LangManager struct {
	lang string
	data map[string]string
}

func NewLangManager(lang string, dataDir string) *LangManager {
	lm := &LangManager{
		lang: lang,
		data: make(map[string]string),
	}

	lm.load(dataDir)
	return lm
}

func DetectLanguage() string {

	// Windows 전용
	if isWindows() {
		return detectWindowsLanguage()
	}

	// Linux / macOS
	langEnv := strings.ToLower(os.Getenv("LANG"))
	if strings.HasPrefix(langEnv, "ko") {
		return "ko"
	}

	return "en"
}

func isWindows() bool {
	return runtime.GOOS == "windows"
}

func (lm *LangManager) load(dataDir string) {
	filePath := filepath.Join(dataDir, lm.lang+".json")

	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return
	}

	if err := json.Unmarshal(bytes, &lm.data); err != nil {
		println("Lang JSON parse error:", err.Error())
	}
}

func (lm *LangManager) loadFile(lang string) bool {
	filePath := filepath.Join("var", "data", lang+".json")
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}
	_ = json.Unmarshal(bytes, &lm.data)
	return true
}

func (lm *LangManager) T(key string) string {
	if val, ok := lm.data[key]; ok {
		return val
	}

	// key 없으면 key 그대로 반환
	return key
}
