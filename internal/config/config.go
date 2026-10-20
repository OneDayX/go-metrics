// Package config holds what the agent and the server configs share.
package config

import (
	"flag"
	"io"
	"os"
)

// FilePath returns the path to the JSON config file: the CONFIG variable, or
// else the -c/-config flag from args. The configs call it before they register
// their own flags, so that the file can give those flags their defaults.
func FilePath(args []string) string {
	if path := os.Getenv("CONFIG"); path != "" {
		return path
	}

	var path string

	// This set knows only -c/-config, the other flags are not its business.
	fs := flag.NewFlagSet("config", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&path, "c", "", "")
	fs.StringVar(&path, "config", "", "")

	for len(args) > 0 {
		// Parse stops at a flag it does not know, after passing it, or at a
		// non-flag argument, such as the value of that flag. Skip the
		// argument it got stuck on and go on.
		_ = fs.Parse(args)
		if rest := fs.Args(); len(rest) < len(args) {
			args = rest
		} else {
			args = args[1:]
		}
	}

	return path
}
