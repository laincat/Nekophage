package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func PreferencesPath(dir string) string { return filepath.Join(dir, "ui.json") }

// Preferences is what the window remembers between launches.
type Preferences struct {
	Source string `json:"source,omitempty"`
	Query  string `json:"query,omitempty"`
	// Folder is the favourite folder kept last, so the next comic goes there
	// without choosing it again.
	Folder string `json:"folder,omitempty"`
}

// ReadPreferences loads what the window remembered. A missing or unreadable
// file is not an error: the window starts empty.
func ReadPreferences(path string) (Preferences, bool) {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return Preferences{}, false
	}
	var s Preferences
	if json.Unmarshal(data, &s) != nil {
		return Preferences{}, false
	}
	return s, true
}

func WritePreferences(path string, s Preferences) {
	data, err := json.Marshal(s)
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, data, 0o600)
}
