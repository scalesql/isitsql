package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/fatih/color"
	"golang.org/x/mod/semver"
)

func main() {
	var all bool
	flag.BoolVar(&all, "all", false, "print all modules instead of just updated")
	flag.Parse()
	if all {
		println("-all...")
	}
	println("running go list...")
	out, err := exec.Command("go", "list", "-json", "-u", "-m", "all").Output()
	if err != nil {
		log.Fatal(err)
	}
	// println(string(out))
	src := bytes.NewReader(out)
	decoder := json.NewDecoder(src)
	for decoder.More() {
		var m Module
		major := false
		if err := decoder.Decode(&m); err != nil {
			log.Fatal(err)
		}
		if !m.Indirect {
			msg := m.Path

			if m.Update != nil {
				msg += fmt.Sprintf(" => %s (%s) => %s (%s)", m.Version, m.Time.Format("2006-01-02"), m.Update.Version, m.Update.Time.Format("2006-01-02"))
				existing := semver.Major(m.Version)
				new := semver.Major(m.Update.Version)
				if existing != new {
					major = true
				}
			}
			if m.Update == nil {
				if all {
					color.Green(msg)
				}
			} else {
				if major {
					color.Red(msg)
				} else {
					color.Yellow(msg)
				}
			}
		}
	}
}

type Module struct {
	Path       string       // module path
	Query      string       // version query corresponding to this version
	Version    string       // module version
	Versions   []string     // available module versions
	Replace    *Module      // replaced by this module
	Time       *time.Time   // time version was created
	Update     *Module      // available update (with -u)
	Main       bool         // is this the main module?
	Indirect   bool         // module is only indirectly needed by main module
	Dir        string       // directory holding local copy of files, if any
	GoMod      string       // path to go.mod file describing module, if any
	GoVersion  string       // go version used in module
	Retracted  []string     // retraction information, if any (with -retracted or -u)
	Deprecated string       // deprecation message, if any (with -u)
	Error      *ModuleError // error loading module
	Origin     any          // provenance of module
	Reuse      bool         // reuse of old module info is safe
}

type ModuleError struct {
	Err string // the error itself
}
