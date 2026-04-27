package watcher

import "github.com/fsnotify/fsnotify"

// FileWatcher is the minimal interface over a filesystem watcher needed by Watcher.
// The production implementation wraps *fsnotify.Watcher; tests supply a fake.
type FileWatcher interface {
	Add(name string) error
	Close() error
	Events() <-chan fsnotify.Event
	Errors() <-chan error
}

// NewFileWatcher returns a FileWatcher backed by fsnotify.
func NewFileWatcher() (FileWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &fsnotifyAdapter{w: w}, nil
}

type fsnotifyAdapter struct {
	w *fsnotify.Watcher
}

func (a *fsnotifyAdapter) Add(name string) error        { return a.w.Add(name) }
func (a *fsnotifyAdapter) Close() error                 { return a.w.Close() }
func (a *fsnotifyAdapter) Events() <-chan fsnotify.Event { return a.w.Events }
func (a *fsnotifyAdapter) Errors() <-chan error          { return a.w.Errors }
