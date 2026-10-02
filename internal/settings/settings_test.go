package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Review Focus 5: the user's own settings keep their order and content.
func TestSetStatusLinePreservesOrder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	orig := `{"theme": "dark", "model": "opus", "hooks": {"Stop": [1, 2]}}`
	os.WriteFile(path, []byte(orig), 0o644)
	changed, err := SetStatusLine(dir, "/opt/bin/claude-accounts")
	if err != nil || !changed {
		t.Fatalf("%v %v", changed, err)
	}
	data, _ := os.ReadFile(path)
	text := string(data)
	order := []string{`"theme"`, `"model"`, `"hooks"`, `"statusLine"`}
	last := -1
	for _, k := range order {
		i := strings.Index(text, k)
		if i <= last {
			t.Fatalf("key order changed:\n%s", text)
		}
		last = i
	}
	if !strings.Contains(text, `"command": "\"/opt/bin/claude-accounts\" statusline --base`) {
		t.Fatalf("command missing:\n%s", text)
	}
	if bak, _ := os.ReadFile(path + ".bak"); string(bak) != orig {
		t.Fatalf("backup: %s", bak)
	}
	if changed, err := SetStatusLine(dir, "/opt/bin/claude-accounts"); err != nil || changed {
		t.Fatalf("second run should be a no-op: %v %v", changed, err)
	}
}

func TestSetStatusLineCreatesFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new")
	if _, err := SetStatusLine(dir, "/x/claude-accounts"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "settings.json.bak")); !os.IsNotExist(err) {
		t.Error("nothing to back up")
	}
}

func TestSetStatusLineRejectsBadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	os.WriteFile(path, []byte("{nope"), 0o644)
	if _, err := SetStatusLine(dir, "/x"); err == nil {
		t.Fatal("expected an error")
	}
	if data, _ := os.ReadFile(path); string(data) != "{nope" {
		t.Fatal("a broken file must not be overwritten")
	}
}

func TestRemoveStatusLine(t *testing.T) {
	dir := t.TempDir()
	SetStatusLine(dir, "/x")
	if err := RemoveStatusLine(dir); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "settings.json"))
	if strings.Contains(string(data), "statusLine") {
		t.Fatalf("still there: %s", data)
	}
}
