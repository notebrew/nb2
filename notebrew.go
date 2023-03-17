package nb2

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type FS interface {
	Open(name string) (fs.File, error)
	WriteFile(name string, data []byte, perm fs.FileMode) error
	RemoveAll(name string) error
	List(name string) ([]string, error)
}

type dirFS string

func DirFS(dir string) FS {
	return dirFS(dir)
}

func (dir dirFS) Open(name string) (fs.File, error) {
	return os.Open(filepath.Join(string(dir), name))
}

func (dir dirFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (dir dirFS) RemoveAll(name string) error {
	return os.RemoveAll(filepath.Join(string(dir), name))
}

func (dir dirFS) List(name string) ([]string, error) {
	var names []string
	root := filepath.FromSlash(filepath.Join(string(dir), name)) + string(os.PathSeparator)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			names = append(names, strings.TrimPrefix(path, root))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return names, nil
}

type Notebrew struct {
	Dir FS
}
