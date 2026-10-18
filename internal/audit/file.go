package audit

import (
	"encoding/json"
	"os"
	"sync"
)

// FileObserver writes events to the end of a file, one JSON per line. The file
// stays open until Close. It is safe for concurrent use.
type FileObserver struct {
	mu   sync.Mutex
	file *os.File
}

// NewFileObserver opens the file at path for appending. The file is created
// if it does not exist.
func NewFileObserver(path string) (*FileObserver, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return &FileObserver{file: file}, nil
}

// Notify appends the event to the file.
func (f *FileObserver) Notify(event Event) error {
	// Marshal does not touch the file, so it runs before taking the lock.
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	f.mu.Lock()
	defer f.mu.Unlock()

	_, err = f.file.Write(data)
	return err
}

// Close closes the file.
func (f *FileObserver) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.file.Close()
}
