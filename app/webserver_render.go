package app

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/leekchan/gtf"
	"github.com/scalesql/isitsql/internal/gui"
	"github.com/scalesql/isitsql/static"
	"github.com/sirupsen/logrus"
)

var errorPageHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>IsItSQL Parsing Error</title>
  </head>
  <body>
    <h1>IsItSQL Parsing Error</h1>
	<p>A parsing error occurred.  Please see the error log for details.</p>
	<p><ul>
		<li><a href="/">Home</a></li>
		<li><a href="/log">Application Log</a> - This logs the template name.  The detailed error is in the log folder with the executable.</li>
	</ul></p>
  </body>
</html>
`

// renderFS renders a template with the passed Context
func renderFS(w http.ResponseWriter, tmpl string, context Context) {
	context.Static = STATIC_URL
	renderFSDynamic(w, tmpl, context)
}

// renderFSDynamic renders a template with the passed data
func renderFSDynamic(w http.ResponseWriter, tmpl string, data any) {
	// TODO: cache all the templates
	// https://stackoverflow.com/questions/50842389/parsing-multiple-templates-in-go

	globalConfig.RLock()
	useLocal := globalConfig.AppConfig.UseLocalStatic
	globalConfig.RUnlock()

	baseTemplateString, err := static.ReadFile(useLocal, "templates/base.html")
	if err != nil {
		renderError(w, tmpl, wrap(err, "static.readfile: base.html"))
		return
	}

	baseTemplate, err := template.New(tmpl).Funcs(gtf.GtfFuncMap).Funcs(gui.TemplateFuncs).Parse(baseTemplateString)
	if err != nil {
		renderError(w, tmpl, wrap(err, "template.New: base.html"))
		return
	}

	templateString, err := static.ReadFile(useLocal, fmt.Sprintf("templates/%s.html", tmpl))
	if err != nil {
		renderError(w, tmpl, wrapf(err, "static.readfile: %s", tmpl))
		return
	}
	newt, err := baseTemplate.Clone()
	if err != nil {
		renderError(w, tmpl, wrap(err, "basetemplate.clone"))
		return
	}
	//finalTemplate, err := template.Must(baseTemplate.Clone()).Parse(templateString)
	finalTemplate, err := newt.Parse(templateString)
	if err != nil {
		renderError(w, tmpl, wrap(err, "newt.parse"))
		return
	}

	err = finalTemplate.Execute(w, data)
	if err != nil {
		renderError(w, tmpl, wrap(err, "finaltemplate.execute"))
	}
}

// renderError shows a basic HTML page in case of a template error
func renderError(w http.ResponseWriter, name string, err error) {
	WinLogErr(fmt.Errorf("template error: %s.html", name))
	logrus.Error(wrap(err, name))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	_, werr := w.Write([]byte(errorPageHTML))
	if werr != nil {
		logrus.Error(wrap(werr, "renderError: w.write"))
	}
}

// render is used for some legacy pages
func render(w http.ResponseWriter, tmpl string, context Context) {
	context.Static = STATIC_URL
	renderFS(w, tmpl, context)
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
