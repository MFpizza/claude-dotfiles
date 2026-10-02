package fsutil

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWriteJSONRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.json")
	in := map[string]any{"name": "🐔 & <b>"}
	if err := WriteJSON(path, in, false); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "🐔 & <b>") {
		t.Fatalf("text was escaped: %s", data)
	}
	var out map[string]any
	if err := ReadJSON(path, &out); err != nil || out["name"] != in["name"] {
		t.Fatalf("got %v, %v", out, err)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Fatalf("temp file left behind: %v", entries)
	}
}

func TestWriteFileAtomicKeepsPerm(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no unix permissions")
	}
	path := filepath.Join(t.TempDir(), "secret.json")
	if err := WriteFileAtomic(path, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	fi, _ := os.Stat(path)
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("perm %v", fi.Mode().Perm())
	}
}

func TestReadJSONMissing(t *testing.T) {
	var v map[string]any
	err := ReadJSON(filepath.Join(t.TempDir(), "nope.json"), &v)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("got %v", err)
	}
}
