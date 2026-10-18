package b

import "os"

// main is not in package main, so the call is allowed.
func main() {
	os.Exit(1)
}
