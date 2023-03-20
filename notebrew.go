package nb2

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type FS interface {
	Open(name string) (fs.File, error)
	WriteFile(name string, data []byte, perm fs.FileMode) error
	RemoveAll(name string) error
	WalkDir(root string, fn fs.WalkDirFunc) error
}

type dirFS string

func DirFS(dir string) FS {
	return dirFS(dir)
}

func (dir dirFS) Open(name string) (fs.File, error) {
	return os.Open(filepath.Join(string(dir), name))
}

func (dir dirFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	path := filepath.Join(string(dir), name)
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, perm)
}

func (dir dirFS) RemoveAll(name string) error {
	return os.RemoveAll(filepath.Join(string(dir), name))
}

func (dir dirFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(filepath.Join(string(dir), root), func(name string, d fs.DirEntry, err error) error {
		name = strings.TrimPrefix(strings.TrimPrefix(name, string(dir)), string(os.PathSeparator))
		return fn(name, d, err)
	})
}

type Notebrew struct {
	fsys FS
}

func New(fsys FS) *Notebrew {
	return &Notebrew{fsys: fsys}
}

func (nb *Notebrew) Cleanup() error {
	return nil
}

func (nb *Notebrew) Addr() (addr string, err error) {
	const name = "local_url.txt"
	file, err := nb.fsys.Open(name)
	if err == nil {
		b, err := io.ReadAll(file)
		file.Close()
		if err == nil {
			addr = string(bytes.TrimSpace(b))
			ln, err := net.Listen("tcp", addr)
			if err == nil {
				return addr, ln.Close()
			}
			nb.fsys.RemoveAll(name)
		}
	}
	ln, err := net.Listen("tcp", "127.0.0.1:80")
	if err != nil {
		ln, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return "", err
		}
	}
	defer ln.Close()
	addr = ln.Addr().String()
	err = nb.fsys.WriteFile(name, []byte(addr), 0644)
	if err != nil {
		return "", err
	}
	return addr, ln.Close()
}

// admin
// api
// static
// image
// post
// note

func (nb *Notebrew) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/admin/lib/", nb.ServeFile)
	mux.HandleFunc("/admin/script/", nb.ServeFile)
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

// assets //

var rootFS = os.DirFS(".")

func (nb *Notebrew) ServeFile(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/admin/")
	file, err := rootFS.Open(strings.TrimSuffix(name, "/"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fileinfo, err := file.Stat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if fileinfo.IsDir() {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if strings.HasSuffix(name, ".gz") {
		ext := path.Ext(strings.TrimSuffix(name, ".gz"))
		if ext != "" {
			mimeType := mime.TypeByExtension(ext)
			w.Header().Set("Content-Type", mimeType)
			w.Header().Set("Content-Encoding", "gzip")
		}
	}
	fileseeker, ok := file.(io.ReadSeeker)
	if ok {
		http.ServeContent(w, r, name, fileinfo.ModTime(), fileseeker)
		return
	}
	var buf bytes.Buffer
	buf.Grow(int(fileinfo.Size()))
	_, err = buf.ReadFrom(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, name, fileinfo.ModTime(), bytes.NewReader(buf.Bytes()))
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
