package watcher

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
)

const defaultDebounce = 300 * time.Millisecond

// Watcher watches a set of paths and calls rebuild after a debounce period
// whenever a relevant filesystem event occurs.
type Watcher struct {
	fw       FileWatcher
	paths    []string
	rebuild  func() error
	debounce time.Duration
	out      io.Writer
	errOut   io.Writer
}

// Option configures a Watcher.
type Option func(*Watcher)

// WithDebounce sets the debounce delay between the first event and the rebuild call.
func WithDebounce(d time.Duration) Option {
	return func(w *Watcher) { w.debounce = d }
}

// WithOutput sets the writers for informational and error messages.
func WithOutput(out, errOut io.Writer) Option {
	return func(w *Watcher) {
		w.out = out
		w.errOut = errOut
	}
}

// New creates a Watcher. It registers all paths with fw immediately so that
// callers can detect Add errors before calling Run.
func New(fw FileWatcher, paths []string, rebuild func() error, opts ...Option) (*Watcher, error) {
	w := &Watcher{
		fw:       fw,
		paths:    paths,
		rebuild:  rebuild,
		debounce: defaultDebounce,
		out:      os.Stdout,
		errOut:   os.Stderr,
	}
	for _, o := range opts {
		o(w)
	}

	for _, p := range paths {
		if err := fw.Add(p); err != nil {
			return nil, fmt.Errorf("watching %s: %w", p, err)
		}
	}

	return w, nil
}

// Run blocks until ctx is cancelled or the underlying FileWatcher closes its
// channels. It calls rebuild() after the debounce period whenever a write,
// create, remove, or rename event is received.
func (w *Watcher) Run(ctx context.Context) error {
	var debounce *time.Timer

	for {
		select {
		case <-ctx.Done():
			if debounce != nil {
				debounce.Stop()
			}
			return nil

		case event, ok := <-w.fw.Events():
			if !ok {
				return nil
			}
			if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) &&
				!event.Has(fsnotify.Remove) && !event.Has(fsnotify.Rename) {
				continue
			}
			name := event.Name
			if debounce != nil {
				debounce.Stop()
			}
			debounce = time.AfterFunc(w.debounce, func() {
				fmt.Fprintf(w.out, "Change detected: %s, rebuilding...\n", name)
				if err := w.rebuild(); err != nil {
					fmt.Fprintf(w.errOut, "rebuild error: %v\n", err)
				} else {
					fmt.Fprintln(w.out, "Rebuild complete.")
				}
			})

		case err, ok := <-w.fw.Errors():
			if !ok {
				return nil
			}
			fmt.Fprintf(w.errOut, "watcher error: %v\n", err)
		}
	}
}
