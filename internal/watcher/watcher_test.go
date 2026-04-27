package watcher_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/peacefixation/ssg/internal/watcher"
)

// fakeWatcher lets tests push events and errors directly without touching the filesystem.
type fakeWatcher struct {
	events chan fsnotify.Event
	errors chan error
	added  []string
}

func newFakeWatcher() *fakeWatcher {
	return &fakeWatcher{
		events: make(chan fsnotify.Event, 8),
		errors: make(chan error, 8),
	}
}

func (f *fakeWatcher) Add(name string) error {
	f.added = append(f.added, name)
	return nil
}

func (f *fakeWatcher) Close() error {
	close(f.events)
	close(f.errors)
	return nil
}

func (f *fakeWatcher) Events() <-chan fsnotify.Event { return f.events }
func (f *fakeWatcher) Errors() <-chan error          { return f.errors }

// shortDebounce is used in tests to avoid long waits.
const shortDebounce = 5 * time.Millisecond

func newTestWatcher(t *testing.T, fw *fakeWatcher, rebuild func() error) *watcher.Watcher {
	t.Helper()
	var out, errOut bytes.Buffer
	w, err := watcher.New(fw, []string{"a", "b"}, rebuild,
		watcher.WithDebounce(shortDebounce),
		watcher.WithOutput(&out, &errOut),
	)
	if err != nil {
		t.Fatalf("watcher.New: %v", err)
	}
	return w
}

func TestNew_RegistersPaths(t *testing.T) {
	fw := newFakeWatcher()
	newTestWatcher(t, fw, func() error { return nil })

	if len(fw.added) != 2 || fw.added[0] != "a" || fw.added[1] != "b" {
		t.Errorf("expected paths [a b], got %v", fw.added)
	}
}

func TestRun_CallsRebuildOnWriteEvent(t *testing.T) {
	fw := newFakeWatcher()
	called := make(chan struct{}, 1)
	w := newTestWatcher(t, fw, func() error {
		called <- struct{}{}
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()

	fw.events <- fsnotify.Event{Name: "a/file.md", Op: fsnotify.Write}

	select {
	case <-called:
		// pass
	case <-time.After(time.Second):
		t.Fatal("rebuild was not called within timeout")
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func TestRun_DebouncesRapidEvents(t *testing.T) {
	fw := newFakeWatcher()
	callCount := 0
	w := newTestWatcher(t, fw, func() error {
		callCount++
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()

	// Send several events in quick succession.
	for range 5 {
		fw.events <- fsnotify.Event{Name: "a/file.md", Op: fsnotify.Write}
	}

	// Wait long enough for the debounce to fire.
	time.Sleep(shortDebounce * 10)

	cancel()
	<-done

	if callCount != 1 {
		t.Errorf("expected rebuild called once, got %d", callCount)
	}
}

func TestRun_IgnoresChmodEvents(t *testing.T) {
	fw := newFakeWatcher()
	called := make(chan struct{}, 1)
	w := newTestWatcher(t, fw, func() error {
		called <- struct{}{}
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()

	fw.events <- fsnotify.Event{Name: "a/file.md", Op: fsnotify.Chmod}

	// Give time for a spurious call to appear.
	time.Sleep(shortDebounce * 10)

	select {
	case <-called:
		t.Error("rebuild should not be called for Chmod events")
	default:
		// pass
	}

	cancel()
	<-done
}

func TestRun_LogsRebuildError(t *testing.T) {
	fw := newFakeWatcher()
	var out, errOut bytes.Buffer
	w, err := watcher.New(fw, []string{"a"}, func() error {
		return errors.New("build failed")
	},
		watcher.WithDebounce(shortDebounce),
		watcher.WithOutput(&out, &errOut),
	)
	if err != nil {
		t.Fatalf("watcher.New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()

	fw.events <- fsnotify.Event{Name: "a/file.md", Op: fsnotify.Write}

	time.Sleep(shortDebounce * 10)
	cancel()
	<-done

	if errOut.Len() == 0 {
		t.Error("expected error message in errOut, got nothing")
	}
	if out.String() == "Rebuild complete.\n" {
		t.Error("should not print success message on error")
	}
}

func TestRun_LogsWatcherError(t *testing.T) {
	fw := newFakeWatcher()
	var out, errOut bytes.Buffer
	w, err := watcher.New(fw, []string{"a"}, func() error { return nil },
		watcher.WithDebounce(shortDebounce),
		watcher.WithOutput(&out, &errOut),
	)
	if err != nil {
		t.Fatalf("watcher.New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()

	fw.errors <- errors.New("inotify limit reached")

	time.Sleep(shortDebounce * 10)
	cancel()
	<-done

	if errOut.Len() == 0 {
		t.Error("expected watcher error in errOut, got nothing")
	}
}

func TestRun_StopsOnContextCancel(t *testing.T) {
	fw := newFakeWatcher()
	w := newTestWatcher(t, fw, func() error { return nil })

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error on cancel: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not return after context cancel")
	}
}

func TestRun_StopsWhenEventChannelClosed(t *testing.T) {
	fw := newFakeWatcher()
	w := newTestWatcher(t, fw, func() error { return nil })

	ctx := context.Background()
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()

	fw.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error on channel close: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not return after channel close")
	}
}
