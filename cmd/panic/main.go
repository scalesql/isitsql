package main

import "github.com/scalesql/isitsql/internal/failure"

func main() {
	defer failure.HandlePanic()
	panic("my panic")
}
