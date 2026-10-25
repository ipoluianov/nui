package ui

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// Settings are the settings of an application, kept between its starts in
// a JSON file of the user's configuration directory (~/.config/<app> on
// Linux, %AppData%\<app> on Windows, ~/Library/Application Support/<app>
// on macOS). Any value JSON encodes goes in: numbers, strings, structs.
//
//	settings, _ := ui.OpenSettings("myapp")
//	theme := settings.String("theme", "light")
//	settings.Set("theme", "dark")
//	settings.Save()
//
// Safe for concurrent use.
type Settings struct {
	mu     sync.Mutex
	path   string
	values map[string]json.RawMessage
}

// SettingsDir returns the directory of the application's settings and
// other files of its own, made if missing
func SettingsDir(app string) (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, app)
	return dir, os.MkdirAll(dir, 0o755)
}

// OpenSettings reads the settings of the application (none yet is no
// error). On an error the settings are empty and work in memory, so the
// application runs without them rather than not at all.
func OpenSettings(app string) (*Settings, error) {
	s := &Settings{values: map[string]json.RawMessage{}}
	dir, err := SettingsDir(app)
	if err != nil {
		return s, err
	}
	return s, s.load(filepath.Join(dir, "settings.json"))
}

// OpenSettingsFile reads the settings from the file, e.g. a portable
// installation's next to the program
func OpenSettingsFile(path string) (*Settings, error) {
	s := &Settings{values: map[string]json.RawMessage{}}
	return s, s.load(path)
}

func (s *Settings) load(path string) error {
	s.path = path
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.values)
}

// Get decodes the value of the key into v (a pointer); false if there is
// no such value or it doesn't fit v
func (s *Settings) Get(key string, v any) bool {
	s.mu.Lock()
	raw, ok := s.values[key]
	s.mu.Unlock()
	return ok && json.Unmarshal(raw, v) == nil
}

// Set keeps the value for the key; Save writes it
func (s *Settings) Set(key string, v any) {
	raw, err := json.Marshal(v)
	if err != nil {
		return
	}
	s.mu.Lock()
	s.values[key] = raw
	s.mu.Unlock()
}

// Delete forgets the key
func (s *Settings) Delete(key string) {
	s.mu.Lock()
	delete(s.values, key)
	s.mu.Unlock()
}

// String returns the string of the key, def if there is none
func (s *Settings) String(key, def string) string {
	v := def
	s.Get(key, &v)
	return v
}

// Int returns the integer of the key, def if there is none
func (s *Settings) Int(key string, def int) int {
	v := def
	s.Get(key, &v)
	return v
}

// Float returns the number of the key, def if there is none
func (s *Settings) Float(key string, def float64) float64 {
	v := def
	s.Get(key, &v)
	return v
}

// Bool returns the flag of the key, def if there is none
func (s *Settings) Bool(key string, def bool) bool {
	v := def
	s.Get(key, &v)
	return v
}

// Save writes the settings to their file: to a temporary file first, then
// renamed, so a crash doesn't leave half of them
func (s *Settings) Save() error {
	if s.path == "" {
		return errors.New("nui: the settings have no file")
	}
	s.mu.Lock()
	data, err := json.MarshalIndent(s.values, "", "  ")
	s.mu.Unlock()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// Path returns the file of the settings
func (s *Settings) Path() string {
	return s.path
}
