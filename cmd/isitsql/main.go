//go:generate esc -prefix=static -pkg="static"  -o=embed_static.go static
//go:generate goversioninfo -64=true

package main

import (
	"fmt"
	"os"

	"github.com/scalesql/isitsql/app"
	"github.com/scalesql/isitsql/internal/failure"
	"github.com/bugsnag/panicwrap"
)

var buildGit = "undefined"
var buildDate = "undefined"

func main() {
	exitStatus, err := panicwrap.BasicWrap(panicHandler)
	if err != nil {
		// Something went wrong setting up the panic wrapper. Unlikely,
		// but possible.
		panic(err)
	}

	// If exitStatus >= 0, then we're the parent process and the panicwrap
	// re-executed ourselves and completed. Just exit with the proper status.
	if exitStatus >= 0 {
		os.Exit(exitStatus)
	}

	// Otherwise, exitStatus < 0 means we're the child. Continue executing as
	// normal...
	app.Run(buildGit, buildDate)
}

func panicHandler(output string) {
	fmt.Fprintln(os.Stderr, output)
	failure.WriteFile("panic", output)
	//os.Exit(1)
}
