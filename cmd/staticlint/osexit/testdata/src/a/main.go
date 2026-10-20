package main

import (
	"os"
	sys "os"
)

func main() {
	defer cleanup()

	if len(os.Args) > 1 {
		os.Exit(2) // want "direct call to os.Exit in the main function of the main package"
	}

	func() {
		sys.Exit(1) // want "direct call to os.Exit in the main function of the main package"
	}()

	exit(0)
}

// exit is not main, so os.Exit is allowed here.
func exit(code int) {
	os.Exit(code)
}

func cleanup() {}
