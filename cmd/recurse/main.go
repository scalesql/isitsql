package main

import (
	"fmt"

	"github.com/scalesql/isitsql/internal/failure"
)

func main() {
	println("starting...")
	defer func() {
		fmt.Println("one")
	}()
	defer failure.HandlePanic()
	incr(1)
	println("done")
}

func incr(i int) int {
	defer func() {
		fmt.Println("one")
	}()
	return incr(i)
}
