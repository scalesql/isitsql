package main

import "github.com/scalesql/isitsql/internal/logring"

func main() {
	r := logring.New(5)
	r.Display()
	for char := 'a'; char <= 'z'; char++ {
		r.Enqueue(string(char))
		r.Display()
	}
}
