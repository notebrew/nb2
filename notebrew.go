package nb2

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"text/template"
)

type FileSystem interface {
	Open(name string) (fs.File, error)
	OpenWriter(name string) (io.WriteCloser, error)
	RemoveAll(path string) error
	WalkDir(root string, fn fs.WalkDirFunc) error
}

// RemoveAll

type dirFS string

func DirFS(dir string) FileSystem {
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

type Mode int

const (
	ModeLocalhost = iota
	ModeSinglesite
	ModeMultisite
)

var modeNames = []string{
	ModeLocalhost:  "localhost",
	ModeSinglesite: "singlesite",
	ModeMultisite:  "multisite",
}

func (m Mode) Enumerate() []string {
	return modeNames
}

type Config struct {
	Dialect   string
	DB        *sql.DB
	FS        FileSystem
	Mode      Mode
	NewServer func(addr string, h http.Handler) *http.Server
}

type Notebrew struct {
	dialect string
	db      *sql.DB
	fs      FileSystem
	mode    Mode
	stop    chan os.Signal
}

func New(c Config) (*Notebrew, error) {
	nb := &Notebrew{
		dialect: c.Dialect,
		db:      c.DB,
		fs:      c.FS,
		mode:    c.Mode,
		stop:    make(chan os.Signal, 1),
	}
	signal.Notify(nb.stop, syscall.SIGINT, syscall.SIGTERM)
	return nb, nil
}

func (nb *Notebrew) isResource(s string) bool {
	switch s {
	case "static", "image", "template", "post", "page", "note":
		return true
	default:
		return false
	}
}

func (nb *Notebrew) Cleanup() error {
	if nb.db == nil {
		return nil
	}
	switch nb.dialect {
	case "sqlite":
		nb.db.Exec(`PRAGMA analysis_limit(400); PRAGMA optimize;`)
	}
	return nb.db.Close()
}

func (nb *Notebrew) Addr() (addr string, err error) {
	const name = "notebrew.url"
	file, err := nb.fs.Open(name)
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
			nb.fs.RemoveAll(name)
		}
	}
	ln, err := net.Listen("tcp", "127.0.0.1:2048")
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
	// TODO: If we're doing "open url if port 3030 already running" then we
	// don't need a notebrew.url file anymore.
	data := "[InternetShortcut]" + newline + "URL=http://" + addr + newline
	writer, err := nb.fs.OpenWriter(name)
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

func (nb *Notebrew) Serve() {
}

func callermsg(a ...any) string {
	_, file, line, _ := runtime.Caller(1)
	var b strings.Builder
	b.WriteString(file + ":" + strconv.Itoa(line))
	for _, v := range a {
		b.WriteString("\n" + fmt.Sprint(v))
	}
	return b.String()
}

func (nb *Notebrew) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/create/")
	head, tail, found := strings.Cut(path, "/")
	if !found {
		http.NotFound(w, r)
		return
	}
	var domain string
	if strings.LastIndex(head, ".") > 0 {
		domain, path = head, tail
	}
	head, tail, found = strings.Cut(path, "/")
	if !found {
		http.NotFound(w, r)
		return
	}
	if !nb.isResource(head) {
		http.Error(w, fmt.Sprintf("invalid resource type %q", head), http.StatusUnprocessableEntity)
		return
	}
	// /api/<action>/<domain>/<resource>/<id>
	_ = domain
}

func (nb *Notebrew) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/admin/lib/", nb.ServeFile)
	mux.HandleFunc("/admin/script/", nb.ServeFile)
	mux.HandleFunc("/api/create/", nb.Create)
	mux.HandleFunc("/api/update/", http.NotFound)
	mux.HandleFunc("/api/delete/", http.NotFound)
	mux.HandleFunc("/api/rename/", http.NotFound)
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:  "s",
			Value: "1",
		})
		http.Redirect(w, r, "https://google.com/", http.StatusFound)
	})
	mux.HandleFunc("/", nb.Page)
	mux.HandleFunc("/static/", nb.Static)
	mux.HandleFunc("/image/", nb.Image)
	mux.HandleFunc("/post/", nb.Post)
	mux.HandleFunc("/note/", nb.Note)
	mux.HandleFunc("/admin/static/", nb.StaticAdmin)
	mux.HandleFunc("/admin/image/", nb.ImageAdmin)
	mux.HandleFunc("/admin/template/", nb.TemplateAdmin)
	mux.HandleFunc("/admin/post/", nb.PostAdmin)
	mux.HandleFunc("/admin/page/", nb.PageAdmin)
	return mux
}

// assets //

var rootFS = os.DirFS(".")

func (nb *Notebrew) ServeFile(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/admin/"), "/")
	file, err := rootFS.Open(name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		http.Error(w, callermsg(err), http.StatusInternalServerError)
		return
	}
	fileinfo, err := file.Stat()
	if err != nil {
		http.Error(w, callermsg(err), http.StatusInternalServerError)
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
		http.Error(w, callermsg(err), http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, name, fileinfo.ModTime(), bytes.NewReader(buf.Bytes()))
}

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

func (nb *Notebrew) TemplateAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

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

func (nb *Notebrew) Note(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}
