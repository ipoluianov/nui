package ui

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ipoluianov/nui/internal/platforms"
)

// A second start passes its arguments to the running copy exactly: a line
// break in an argument (legal in a Unix file name) used to end it early or
// pass for the "show" command, and an argument over 64 KB lost the whole
// start
func TestInstanceArgsArrive(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	dir := t.TempDir()
	first, ok := acquireSingleInstance(dir, nil)
	if !ok || first.listener == nil {
		t.Skip("no local socket")
	}
	defer first.Close()
	got := make(chan []string, 4)
	first.OnActivate(func(args []string) { got <- args })
	for _, args := range [][]string{
		{"/tmp/a\nshow", "/tmp/b"},
		{"/tmp/x\ny.txt"},
		{strings.Repeat("a", 70000)},
		{"arg \"q\"", "\\", "bad utf8 \xff", ""},
		nil,
	} {
		if _, ok := acquireSingleInstance(dir, args); ok {
			t.Fatal("the second start became the first")
		}
		var a []string
		delivered := false
		for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline) && !delivered; {
			platforms.RunHeadlessPosted()
			select {
			case a = <-got:
				delivered = true
			default:
				time.Sleep(5 * time.Millisecond)
			}
		}
		if !delivered {
			t.Errorf("%.40q: not delivered", args)
			continue
		}
		if len(a) != len(args) || (len(a) > 0 && !reflect.DeepEqual(a, args)) {
			t.Errorf("sent %.40q, got %.40q", args, a)
		}
	}
}

// Whatever else writes to the socket is not taken for a start
func TestInstanceGarbageIgnored(t *testing.T) {
	for _, msg := range []string{
		"arg /unquoted\nshow\n",
		"arg \"/a\"\n",
		"arg \"/a\"\nbogus\n",
		"show-me\n",
		strings.Repeat("x", 100000) + "\nshow\n",
	} {
		client, server := net.Pipe()
		go func() {
			client.Write([]byte(msg))
			client.Close()
		}()
		if args, ok := readInstanceMessage(server); ok {
			t.Errorf("%.30q: taken for a start with %.30q", msg, args)
		}
		server.Close()
	}
}

// A settings file of "null" used to set the map to nil: every later Set
// panicked. A broken file leaves no half-decoded values either.
func TestSettingsNullAndBrokenFile(t *testing.T) {
	for _, content := range []string{"null", `{"a": 1, "b": [}`, `[1, 2]`} {
		path := filepath.Join(t.TempDir(), "settings.json")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		s, _ := OpenSettingsFile(path)
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%q: Set panics: %v", content, r)
				}
			}()
			s.Set("theme", "dark")
		}()
		if got := s.String("theme", ""); got != "dark" {
			t.Errorf("%q: theme = %q", content, got)
		}
		if got := s.Int("a", -1); got != -1 {
			t.Errorf("%q: a half-decoded value is kept: a = %d", content, got)
		}
	}
}

// Settings are safe for concurrent use, Save too: saves at once used to
// share one temporary file, and some failed
func TestSettingsConcurrentSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s, _ := OpenSettingsFile(path)
	for i := 0; i < 200; i++ {
		s.Set(fmt.Sprint("k", i), strings.Repeat("v", 1000))
	}
	var failed sync.Map
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 30; i++ {
				if err := s.Save(); err != nil {
					failed.Store(err.Error(), true)
				}
			}
		}()
	}
	wg.Wait()
	failed.Range(func(k, _ any) bool {
		t.Error("Save:", k)
		return true
	})
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil || len(m) != 200 {
		t.Errorf("the file: %d keys, %v", len(m), err)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Errorf("temporary files are left: %d entries", len(entries))
	}
}
