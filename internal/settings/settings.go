// Package settings edits the statusLine entry of Claude Code's settings.json while
// leaving the user's other settings, and their order, exactly as they were.
package settings

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/MFpizza/claude-dotfiles/internal/fsutil"
)

// Command is the status line command. Forward slashes work in every shell Claude
// Code uses, including on Windows.
func Command(exe, mainDir string) string {
	return fmt.Sprintf(`"%s" statusline --base "%s"`, filepath.ToSlash(exe), filepath.ToSlash(mainDir))
}

func SetStatusLine(mainDir, exe string) (bool, error) {
	path, data, obj, err := load(mainDir)
	if err != nil {
		return false, err
	}
	want := Command(exe, mainDir)
	var cur struct {
		Command string `json:"command"`
	}
	if raw, ok := obj.vals["statusLine"]; ok && json.Unmarshal(raw, &cur) == nil && cur.Command == want {
		return false, nil
	}
	val, _ := json.Marshal(map[string]any{"type": "command", "command": want, "padding": 0})
	obj.set("statusLine", val)
	return true, save(path, data, obj)
}

func RemoveStatusLine(mainDir string) error {
	path, data, obj, err := load(mainDir)
	if err != nil {
		return err
	}
	if _, ok := obj.vals["statusLine"]; !ok {
		return nil
	}
	obj.del("statusLine")
	return save(path, data, obj)
}

func load(mainDir string) (string, []byte, *object, error) {
	path := filepath.Join(mainDir, "settings.json")
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return path, nil, nil, err
	}
	obj, err := parse(data)
	if err != nil {
		return path, nil, nil, fmt.Errorf("%s: %w", path, err)
	}
	return path, data, obj, nil
}

// save keeps the previous file as settings.json.bak before writing.
func save(path string, old []byte, obj *object) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if old != nil {
		if err := os.WriteFile(path+".bak", old, 0o644); err != nil {
			return err
		}
	}
	return fsutil.WriteFileAtomic(path, obj.marshal(), 0o644)
}

// object is a JSON object that remembers its key order.
type object struct {
	keys []string
	vals map[string]json.RawMessage
}

func parse(data []byte) (*object, error) {
	obj := &object{vals: map[string]json.RawMessage{}}
	if len(bytes.TrimSpace(data)) == 0 {
		return obj, nil
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return nil, errors.New("not a JSON object")
	}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, err
		}
		obj.set(tok.(string), raw)
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	return obj, nil
}

func (o *object) set(key string, val json.RawMessage) {
	if _, ok := o.vals[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.vals[key] = val
}

func (o *object) del(key string) {
	delete(o.vals, key)
	for i, k := range o.keys {
		if k == key {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			return
		}
	}
}

func (o *object) marshal() []byte {
	var b bytes.Buffer
	b.WriteString("{")
	for i, k := range o.keys {
		if i > 0 {
			b.WriteString(",")
		}
		key, _ := json.Marshal(k)
		b.WriteString("\n  ")
		b.Write(key)
		b.WriteString(": ")
		if err := json.Indent(&b, o.vals[k], "  ", "  "); err != nil {
			b.Write(o.vals[k])
		}
	}
	if len(o.keys) > 0 {
		b.WriteString("\n")
	}
	b.WriteString("}\n")
	return b.Bytes()
}
