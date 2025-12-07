package static

import (
	"embed"
	"net/http"
	"os"
	"path/filepath"
)

//go:embed templates css js img fonts docs
var fs embed.FS

// HttpFS returns either an embedded file system or a reference to the local ./static folder.
// Sample: group.Handle("/static/", http.StripPrefix("/static/", http.FileServer(static.HttpFS(useLocal))))
func HttpFS(useLocal bool) http.FileSystem {
	if useLocal {
		return http.Dir("./static")
	} else {
		return http.FS(fs)
	}
}

// FS returns the embedded file system
func FS() embed.FS {
	return fs
}

// ReadFile reads the embedded file system or the local file
// system depending on the flag.
func ReadFile(useLocal bool, name string) (string, error) {
	if useLocal {
		localName := filepath.Join("static", name)
		bb, err := os.ReadFile(localName)
		return string(bb), err
	} else {
		bb, err := fs.ReadFile(name)
		return string(bb), err
	}
}
