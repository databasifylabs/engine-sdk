package nodefs

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
)

const EngineUID = 64999

type Root string

func (r Root) Path(relative string) string {
	return filepath.Join(string(r), relative)
}

func (r Root) Exists(relative string) bool {
	_, err := os.Stat(r.Path(relative))

	return err == nil
}

func (r Root) Read(relative string) string {
	data, err := os.ReadFile(r.Path(relative))
	if err != nil {
		return ""
	}

	return string(data)
}

func (r Root) Empty(relative string) bool {
	entries, err := os.ReadDir(r.Path(relative))

	return err != nil || len(entries) == 0
}

func (r Root) Dir(relative string) error {
	path := r.Path(relative)
	if err := os.MkdirAll(path, 0o750); err != nil {
		return err
	}

	return own(path)
}

func (r Root) Write(relative string, content string, mode os.FileMode, engineOwned bool) (bool, error) {
	path := r.Path(relative)
	if current, err := os.ReadFile(path); err == nil && bytes.Equal(current, []byte(content)) {
		return false, nil
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, []byte(content), mode); err != nil {
		return false, err
	}
	if engineOwned {
		if err := own(temporary); err != nil {
			return false, err
		}
	}

	return true, os.Rename(temporary, path)
}

func own(path string) error {
	err := os.Lchown(path, EngineUID, EngineUID)
	if errors.Is(err, os.ErrPermission) && os.Geteuid() != 0 {
		return nil
	}

	return err
}
