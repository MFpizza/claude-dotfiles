package accounts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MFpizza/claude-dotfiles/internal/links"
)

func write(t *testing.T, path, text string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestShareCreatesLinks(t *testing.T) {
	root := t.TempDir()
	main, acc := filepath.Join(root, "main"), filepath.Join(root, "acc")
	conflicts, err := Share(main, acc)
	if err != nil || len(conflicts) != 0 {
		t.Fatalf("%v %v", conflicts, err)
	}
	for _, name := range SharedDirs {
		if !links.IsLink(filepath.Join(acc, name)) || !links.Points(filepath.Join(acc, name), filepath.Join(main, name)) {
			t.Errorf("%s is not linked", name)
		}
	}
	if _, err := Share(main, acc); err != nil {
		t.Fatalf("second run: %v", err)
	}
}

func TestShareMergesExistingFolder(t *testing.T) {
	root := t.TempDir()
	main, acc := filepath.Join(root, "main"), filepath.Join(root, "acc")
	write(t, filepath.Join(main, "projects", "p1.txt"), "main")
	write(t, filepath.Join(acc, "projects", "p2.txt"), "acc")
	if conflicts, err := Share(main, acc); err != nil || len(conflicts) != 0 {
		t.Fatalf("%v %v", conflicts, err)
	}
	for _, f := range []string{"p1.txt", "p2.txt"} {
		if _, err := os.Stat(filepath.Join(main, "projects", f)); err != nil {
			t.Errorf("%s missing from main", f)
		}
	}
	if !links.IsLink(filepath.Join(acc, "projects")) {
		t.Error("projects should now be a link")
	}
}

func TestShareKeepsConflicts(t *testing.T) {
	root := t.TempDir()
	main, acc := filepath.Join(root, "main"), filepath.Join(root, "acc")
	write(t, filepath.Join(main, "projects", "x.txt"), "main")
	write(t, filepath.Join(acc, "projects", "x.txt"), "acc")
	conflicts, err := Share(main, acc)
	if err != nil || len(conflicts) != 1 || conflicts[0] != "projects" {
		t.Fatalf("%v %v", conflicts, err)
	}
	if data, _ := os.ReadFile(filepath.Join(main, "projects", "x.txt")); string(data) != "main" {
		t.Error("main's file was overwritten")
	}
	if links.IsLink(filepath.Join(acc, "projects")) {
		t.Error("a conflicting folder must stay as it is")
	}
}

func TestRemoveDirKeepsMainData(t *testing.T) {
	root := t.TempDir()
	main, acc := filepath.Join(root, "main"), filepath.Join(root, "acc")
	if _, err := Share(main, acc); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(main, "projects", "keep.txt"), "x")
	write(t, filepath.Join(acc, ".credentials.json"), "{}")
	if err := RemoveDir(acc); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(acc); !os.IsNotExist(err) {
		t.Error("account dir still exists")
	}
	if _, err := os.Stat(filepath.Join(main, "projects", "keep.txt")); err != nil {
		t.Error("main account data was deleted")
	}
	if err := RemoveDir(acc); err != nil {
		t.Errorf("removing a missing dir: %v", err)
	}
}
