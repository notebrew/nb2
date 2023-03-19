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

func (nb *Notebrew) Handler() http.Handler {
	mux := http.NewServeMux()
	// static
	mux.HandleFunc("/static/", nil)           // GET
	mux.HandleFunc("/admin/static/", nil)     // GET
	mux.HandleFunc("/api/static/create", nil) // POST
	mux.HandleFunc("/api/static/delete", nil) // POST
	mux.HandleFunc("/api/static/update", nil) // POST
	// image
	mux.HandleFunc("/image/", nil)           // GET
	mux.HandleFunc("/admin/image/", nil)     // GET
	mux.HandleFunc("/api/image/create", nil) // POST
	mux.HandleFunc("/api/image/delete", nil) // POST
	// template
	mux.HandleFunc("/admin/template/", nil)     // GET
	mux.HandleFunc("/api/template/create", nil) // POST
	mux.HandleFunc("/api/template/delete", nil) // POST
	mux.HandleFunc("/api/template/update", nil) // POST
	mux.HandleFunc("/api/template/rename", nil) // POST
	// post
	mux.HandleFunc("/post/", nil)           // GET
	mux.HandleFunc("/admin/post/", nil)     // GET
	mux.HandleFunc("/api/post/create", nil) // POST
	mux.HandleFunc("/api/post/delete", nil) // POST
	mux.HandleFunc("/api/post/update", nil) // POST
	// note
	mux.HandleFunc("/note/", nil)           // GET
	mux.HandleFunc("/api/note/create", nil) // POST
	mux.HandleFunc("/api/note/delete", nil) // POST
	mux.HandleFunc("/api/note/update", nil) // POST
	return mux
}
