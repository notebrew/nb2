package nb2

import (
	"io/fs"
	"net"
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
	err := os.MkdirAll(filepath.Dir(name), 0755)
	if err != nil {
		return err
	}
	return os.WriteFile(name, data, 0644)
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

type Notebrew struct {
	fsys FS
}

func New(fsys FS) *Notebrew {
	return &Notebrew{fsys: fsys}
}

func (nb *Notebrew) Init() error {
	return nil
}

func (nb *Notebrew) Listener() (net.Listener, error) {
	return nil, nil
}

func (nb *Notebrew) Router() http.Handler {
	mux := http.NewServeMux()
	// static
	mux.HandleFunc("/static/", nb.Static)
	mux.HandleFunc("/admin/static/", nb.StaticAdmin)
	mux.HandleFunc("/api/static/create", nb.StaticCreate)
	mux.HandleFunc("/api/static/delete", nb.StaticDelete)
	mux.HandleFunc("/api/static/update", nb.StaticUpdate)
	mux.HandleFunc("/api/static/rename", nb.StaticRename)
	// image
	mux.HandleFunc("/image/", nb.Image)
	mux.HandleFunc("/admin/image/", nb.ImageAdmin)
	mux.HandleFunc("/api/image/create", nb.ImageCreate)
	mux.HandleFunc("/api/image/delete", nb.ImageDelete)
	// template
	mux.HandleFunc("/admin/template/", nb.TemplateAdmin)
	mux.HandleFunc("/api/template/create", nb.TemplateCreate)
	mux.HandleFunc("/api/template/delete", nb.TemplateDelete)
	mux.HandleFunc("/api/template/update", nb.TemplateUpdate)
	mux.HandleFunc("/api/template/rename", nb.TemplateRename)
	// post
	mux.HandleFunc("/post/", nb.Post)
	mux.HandleFunc("/admin/post/", nb.PostAdmin)
	mux.HandleFunc("/api/post/create", nb.PostCreate)
	mux.HandleFunc("/api/post/delete", nb.PostDelete)
	mux.HandleFunc("/api/post/update", nb.PostUpdate)
	// page
	mux.HandleFunc("/", nb.Page)
	mux.HandleFunc("/admin/page/", nb.PageAdmin)
	mux.HandleFunc("/api/page/create", nb.PageCreate)
	mux.HandleFunc("/api/page/delete", nb.PageDelete)
	mux.HandleFunc("/api/page/update", nb.PageUpdate)
	mux.HandleFunc("/api/page/rename", nb.PageRename)
	// note
	mux.HandleFunc("/note/", nb.Note)
	mux.HandleFunc("/api/note/create", nb.NoteCreate)
	mux.HandleFunc("/api/note/delete", nb.NoteDelete)
	mux.HandleFunc("/api/note/update", nb.NoteUpdate)
	return mux
}

// static //

func (nb *Notebrew) Static(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (nb *Notebrew) StaticAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (nb *Notebrew) StaticCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (nb *Notebrew) StaticDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (nb *Notebrew) StaticUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (nb *Notebrew) StaticRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

// image //

func (nb *Notebrew) Image(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (nb *Notebrew) ImageAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (nb *Notebrew) ImageCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (nb *Notebrew) ImageDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

// template //

func (nb *Notebrew) TemplateAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (nb *Notebrew) TemplateCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (nb *Notebrew) TemplateDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (nb *Notebrew) TemplateUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (nb *Notebrew) TemplateRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

// post //

func (nb *Notebrew) Post(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (nb *Notebrew) PostAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (nb *Notebrew) PostCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (nb *Notebrew) PostDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (nb *Notebrew) PostUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

// page //

func (nb *Notebrew) Page(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (nb *Notebrew) PageAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (nb *Notebrew) PageCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (nb *Notebrew) PageDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (nb *Notebrew) PageUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (nb *Notebrew) PageRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

// note //

func (nb *Notebrew) Note(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (nb *Notebrew) NoteCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (nb *Notebrew) NoteDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

func (nb *Notebrew) NoteUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}
