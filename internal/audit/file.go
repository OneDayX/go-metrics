package audit

import (
	"encoding/json"
	"os"
)

// FileObserver writes events to the end of a file, one JSON per line.
type FileObserver struct {
	path string
}

// NewFileObserver returns an observer that writes to the file at path. The
// file is created on the first event if it does not exist.
func NewFileObserver(path string) *FileObserver {
	return &FileObserver{path: path}
}

// Notify appends the event to the file.
func (f *FileObserver) Notify(event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(f.path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(append(data, '\n'))
	return err
}
