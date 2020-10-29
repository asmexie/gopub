package netserve

import (
	"net/http"
	"os"
	"path"
	"strings"
)

type fileHandler struct {
	root            http.FileSystem
	notfoundHandler http.Handler
}

// FileServer ...
func FileServer(root http.FileSystem, notfoundHandler http.Handler) http.Handler {
	return &fileHandler{
		root:            root,
		notfoundHandler: notfoundHandler,
	}
}

func (fh *fileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
		r.URL.Path = upath
	}
	fh.serveFile(w, r, fh.root, path.Clean(upath), true)
}

func (fh *fileHandler) toHTTPError(err error) (msg string, httpStatus int) {
	if os.IsNotExist(err) {
		return "404 page not found", http.StatusNotFound
	}
	if os.IsPermission(err) {
		return "403 Forbidden", http.StatusForbidden
	}
	// Default:
	return "500 Internal Server Error", http.StatusInternalServerError
}

func localRedirect(w http.ResponseWriter, r *http.Request, newPath string) {
	if q := r.URL.RawQuery; q != "" {
		newPath += "?" + q
	}
	w.Header().Set("Location", newPath)
	w.WriteHeader(http.StatusMovedPermanently)
}

func (fh *fileHandler) writeHTTPFileErr(w http.ResponseWriter, r *http.Request, err error) {
	msg, code := fh.toHTTPError(err)
	if code == http.StatusNotFound && fh.notfoundHandler != nil {
		fh.notfoundHandler.ServeHTTP(w, r)
	} else {
		http.Error(w, msg, code)
	}
}

func (fh *fileHandler) serveFile(w http.ResponseWriter, r *http.Request, fs http.FileSystem, name string, redirect bool) {
	const indexPage = "/index.html"

	// redirect .../index.html to .../
	// can't use Redirect() because that would make the path absolute,
	// which would be a problem running under StripPrefix
	if strings.HasSuffix(r.URL.Path, indexPage) {
		localRedirect(w, r, "./")
		return
	}

	f, err := fs.Open(name)
	if err != nil {
		fh.writeHTTPFileErr(w, r, err)
		return
	}
	defer f.Close()

	d, err := f.Stat()
	if err != nil {
		fh.writeHTTPFileErr(w, r, err)
		return
	}

	if redirect {
		// redirect to canonical path: / at end of directory url
		// r.URL.Path always begins with /
		url := r.URL.Path
		if d.IsDir() {
			if url[len(url)-1] != '/' {
				localRedirect(w, r, path.Base(url)+"/")
				return
			}
		} else {
			if url[len(url)-1] == '/' {
				localRedirect(w, r, "../"+path.Base(url))
				return
			}
		}
	}

	// redirect if the directory name doesn't end in a slash
	if d.IsDir() {
		url := r.URL.Path
		if url[len(url)-1] != '/' {
			localRedirect(w, r, path.Base(url)+"/")
			return
		}
	}

	// use contents of index.html for directory, if present
	if d.IsDir() {
		index := strings.TrimSuffix(name, "/") + indexPage
		ff, err := fs.Open(index)
		if err == nil {
			defer ff.Close()
			dd, err := ff.Stat()
			if err == nil {
				name = index
				d = dd
				f = ff
			}
		}
	}

	if d.IsDir() {
		fh.writeHTTPFileErr(w, r, os.ErrNotExist)
		return
	}

	http.ServeContent(w, r, d.Name(), d.ModTime(), f)
}
