package nb2

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

type FS interface {
	Open(name string) (fs.File, error)
	OpenWriter(name string) (io.WriteCloser, error)
	RemoveAll(path string) error
	WalkDir(root string, fn fs.WalkDirFunc) error
}

// RemoveAll

type dirFS string

func DirFS(dir string) FS {
	return dirFS(dir)
}

func (dir dirFS) Open(name string) (fs.File, error) {
	return os.Open(filepath.Join(string(dir), name))
}

func (dir dirFS) OpenWriter(name string) (io.WriteCloser, error) {
	path := filepath.Join(string(dir), name)
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return nil, err
	}
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
}

func (dir dirFS) RemoveAll(path string) error {
	return os.RemoveAll(filepath.Join(string(dir), path))
}

func (dir dirFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(filepath.Join(string(dir), root), func(name string, d fs.DirEntry, err error) error {
		name = strings.TrimPrefix(name, string(dir))
		name = strings.TrimPrefix(name, string(os.PathSeparator))
		return fn(name, d, err)
	})
}

type Notebrew struct {
	fsys FS
	tlds map[string]struct{}
}

func New(fsys FS) (*Notebrew, error) {
	nb := &Notebrew{
		fsys: fsys,
		tlds: make(map[string]struct{}),
	}
	file, err := rootFS.Open("embed/tlds-alpha-by-domain.txt")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		line = strings.ToLower(strings.TrimSpace(line))
		if line != "" && !strings.HasPrefix(line, "#") {
			nb.tlds[line] = struct{}{}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	return nb, nil
}

func (nb *Notebrew) isDomain(s string) bool {
	i := strings.LastIndex(s, ".")
	if i < 0 {
		return false
	}
	_, ok := nb.tlds[s[i+1:]]
	return ok
}

func (nb *Notebrew) Cleanup() error {
	return nil
}

func (nb *Notebrew) Addr() (addr string, err error) {
	const name = "notebrew.url"
	file, err := nb.fsys.Open(name)
	if err == nil {
		b, err := io.ReadAll(file)
		file.Close()
		if err == nil {
			data := string(b)
			_, data, found := strings.Cut(data, "\nURL")
			if found {
				data = strings.TrimSpace(data)
				data = strings.TrimPrefix(data, "=")
				data = strings.TrimSpace(data)
				addr, _, _ = strings.Cut(data, "\n")
				if strings.HasPrefix(addr, "http://") {
					addr = strings.TrimPrefix(addr, "http://")
				} else if strings.HasPrefix(addr, "https://") {
					addr = strings.TrimPrefix(addr, "https://")
				}
				ln, err := net.Listen("tcp", addr)
				if err == nil {
					return addr, ln.Close()
				}
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
	newline := "\n"
	if runtime.GOOS == "windows" {
		newline = "\r\n"
	}
	data := "[InternetShortcut]" + newline + "URL=http://" + addr + newline
	writer, err := nb.fsys.OpenWriter(name)
	if err != nil {
		return "", err
	}
	defer writer.Close()
	_, err = writer.Write([]byte(data))
	if err != nil {
		return "", err
	}
	err = writer.Close()
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
	mux.HandleFunc("/api/create/", http.NotFound) // TODO
	mux.HandleFunc("/api/update/", http.NotFound) // TODO
	mux.HandleFunc("/api/delete/", http.NotFound) // TODO
	mux.HandleFunc("/api/rename/", http.NotFound) // TODO
	// static | image | template | post | page | note
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
	tmpl, err := template.ParseFS(rootFS, "embed/post.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = buf.WriteTo(w)
	if err != nil {
		log.Println(err)
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
	tmpl, err := template.ParseFS(rootFS, "embed/editor.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = buf.WriteTo(w)
	if err != nil {
		log.Println(err)
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
