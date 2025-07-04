package gui

import (
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"sync"

	"github.com/scalesql/isitsql/static"
	"github.com/leekchan/gtf"
)

// mux protects the local variable
var mux sync.RWMutex

// useLocalTemplates indicates whether to use the file system
// or the embedded file system. The default is false
// which will use the embedded templates.
var useLocalTemplates bool

// SetUseLocal sets a flag to use the local file system
// instead of the embedded templates
func SetUseLocal(useLocal bool) {
	mux.Lock()
	useLocalTemplates = useLocal
	mux.Unlock()
}

func UseLocal() bool {
	mux.RLock()
	defer mux.RUnlock()
	return useLocalTemplates
}

// Execute parses and executes a template.  It takes a variable number
// of templates which probably includes a base, content and any other
// shared templates.
func ExecuteTemplates(w io.Writer, data any, names ...string) error {
	tmpl, err := parseTemplates(names...)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, data)
}

// parseTemplates parses template files from the local file system
// or the embedded file system.
func parseTemplates(names ...string) (*template.Template, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("no template files")
	}
	// The template needs an actual name.  Just use the first one.
	firstName := filepath.Base(names[0])
	local := UseLocal()
	// local needs "static" prepended to the path of each template file
	if local {
		newNames := make([]string, 0, len(names))
		for _, str := range names {
			newNames = append(newNames, filepath.ToSlash(filepath.Join("static", str)))
		}

		t, err := template.New(firstName).Funcs(gtf.GtfFuncMap).Funcs(TemplateFuncs).ParseFiles(newNames...)
		if err != nil {
			return nil, err
		}
		return t, nil
	}
	return template.New(firstName).Funcs(gtf.GtfFuncMap).Funcs(TemplateFuncs).ParseFS(static.FS(), names...)
}
