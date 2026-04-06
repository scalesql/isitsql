package gui

import (
	"bytes"
	"fmt"
	"html/template"
	"sync"

	"github.com/leekchan/gtf"
	"github.com/scalesql/isitsql/static"
	"github.com/sirupsen/logrus"
)

// mux protects the package variables
var mux sync.RWMutex

// templates holds the parsed templates
var templates = make(map[string]*template.Template)

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

// ExecuteTemplate parses and executes the named template.  It also parses
// the "base" template.  It uses a cached template if one exists.
func ExecuteTemplate(name string, data any) ([]byte, error) {
	var t *template.Template
	var ok bool
	var err error

	mux.RLock()
	t, ok = templates[name]
	useLocal := useLocalTemplates
	mux.RUnlock()
	// if there is no cached template or we are using local templates, parse
	if !ok || useLocal {
		t, err = parseTemplate(name)
		if err != nil {
			return []byte{}, wrap(err, "parsetemplate")
		}
		mux.Lock()
		templates[name] = t
		mux.Unlock()
	}
	var buf bytes.Buffer
	err = t.Execute(&buf, data)
	if err != nil {
		return []byte{}, wrap(err, "t.execute")
	}
	return buf.Bytes(), nil
}

// parseTemplate parses the base template and the named template.
func parseTemplate(name string) (*template.Template, error) {
	logrus.Tracef("parsing: %s", name)
	mux.RLock()
	useLocal := useLocalTemplates
	mux.RUnlock()
	str, err := static.ReadFile(useLocal, "templates/base.html")
	if err != nil {
		return nil, wrap(err, "static.readfile: base.html")
	}

	base, err := template.New(name).Funcs(gtf.GtfFuncMap).Funcs(TemplateFuncs).Parse(str)
	if err != nil {
		return nil, wrap(err, "template.New: base.html")
	}

	str, err = static.ReadFile(useLocal, fmt.Sprintf("templates/%s.html", name))
	if err != nil {
		return nil, wrapf(err, "static.readfile: %s", name)
	}
	final, err := base.Parse(str)
	if err != nil {
		return nil, wrap(err, "base.parse")
	}
	return final, nil
}

func wrap(err error, msg string) error {
	if msg == "" {
		return err
	}
	return fmt.Errorf("%s: %w", msg, err)
}

func wrapf(err error, format string, args ...any) error {
	if format == "" {
		return err
	}
	msg := fmt.Sprintf(format, args...)
	return fmt.Errorf("%s: %w", msg, err)
}
