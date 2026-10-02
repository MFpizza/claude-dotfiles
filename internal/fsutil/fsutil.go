// Package fsutil holds the small file helpers shared by every package.
package fsutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

// ReadJSON decodes the file at path into v.
func ReadJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// Marshal encodes v without HTML escaping, indented with two spaces when indent is true.
func Marshal(v any, indent bool) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if indent {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WriteJSON writes v to path atomically.
func WriteJSON(path string, v any, indent bool) error {
	data, err := Marshal(v, indent)
	if err != nil {
		return err
	}
	return WriteFileAtomic(path, data, 0o644)
}

// WriteFileAtomic writes a temp file named after the PID and renames it over path,
// so several status line processes refreshing at once never see a torn file.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp := fmt.Sprintf("%s.%d.tmp", path, os.Getpid())
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
