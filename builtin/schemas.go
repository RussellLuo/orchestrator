package builtin

import (
	"embed"
	"encoding/json"
	"io/fs"
	"path/filepath"
	"strings"
)

// Schemas for built-in tasks.
var TaskSchemas map[string]map[string]any

//go:embed *.schema.json
var schemaFS embed.FS

func init() {
	TaskSchemas = MustCollectFiles(schemaFS, ".schema.json", nil)
}

func MustCollectFiles(f fs.FS, suffix string, convert func(map[string]any) map[string]any) map[string]map[string]any {
	m := map[string]map[string]any{}
	fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			panic(err)
		}
		if d.IsDir() || !strings.HasSuffix(path, suffix) {
			return nil
		}

		content, err := fs.ReadFile(f, path)
		if err != nil {
			panic(err)
		}

		base := filepath.Base(path)
		typ := strings.TrimSuffix(base, suffix)

		value := MustUnmarshalToMap(content)
		if convert != nil {
			value = convert(value)
		}
		m[typ] = value

		return nil
	})
	return m
}

func MustUnmarshalToMap(data []byte) map[string]any {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		panic(err)
	}
	return m
}
