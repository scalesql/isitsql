package app

import (
	"fmt"
	"net/http"

	"github.com/scalesql/isitsql/internal/gui"
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
func renderFSDynamic(w http.ResponseWriter, name string, data any) {
	bb, err := gui.ExecuteTemplate(name, data)
	if err != nil {
		renderError(w, name, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err = w.Write(bb)
	if err != nil {
		logrus.Error(wrap(err, "renderfsdynamic: w.write"))
	}
}

// renderError shows a basic HTML page for any template error
func renderError(w http.ResponseWriter, name string, err error) {
	WinLogErr(fmt.Errorf("template error: %s.html", name))
	logrus.Error(wrap(err, name))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
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
