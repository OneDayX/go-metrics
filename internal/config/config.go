// Package config holds what the agent and the server configs share.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// FilePath returns the path to the JSON config file: the CONFIG variable, or
// else the -c/-config flag from args. The configs call it before they register
// their own flags, so that the file can give those flags their defaults.
func FilePath(args []string) string {
	if path := os.Getenv("CONFIG"); path != "" {
		return path
	}

	var path string

	// The flag can come with one or two dashes, with its value after "=" or
	// in the next argument. The last one wins, like in the flag package.
	for i := 0; i < len(args); i++ {
		if !strings.HasPrefix(args[i], "-") {
			continue
		}

		name, value, hasValue := strings.Cut(strings.TrimLeft(args[i], "-"), "=")
		if name != "c" && name != "config" {
			continue
		}

		if !hasValue && i+1 < len(args) {
			i++
			value = args[i]
		}
		path = value
	}

	return path
}

// LoadFile reads the JSON config file at path into cfg; keys missing from the
// file keep the values cfg already has. The intervals map file keys to where
// their values go: the file has them as durations like "1s", they are saved
// in whole seconds, the way the flags carry them.
func LoadFile(path string, cfg any, intervals map[string]*int64) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return err
	}

	var file map[string]json.RawMessage
	if err := json.Unmarshal(data, &file); err != nil {
		return err
	}

	for key, seconds := range intervals {
		raw, ok := file[key]
		if !ok {
			continue
		}

		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}

		interval, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		*seconds = int64(interval / time.Second)
	}

	return nil
}
