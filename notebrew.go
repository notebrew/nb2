package nb2

import (
	"io/fs"
	"net/http"
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
	err := filepath.WalkDir(filepath.Join(string(dir), name), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		names = append(names, strings.TrimPrefix(strings.TrimPrefix(path, string(dir)), string(os.PathSeparator)))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return names, nil
}

const (
	PrefixFile  = "/file/"
	PrefixNote  = "/note/"
	PrefixPost  = "/post/"
	PrefixImage = "/image/"
)

type Notebrew struct {
	Dir FS
}

func (nb *Notebrew) File(w http.ResponseWriter, r *http.Request) {
}
