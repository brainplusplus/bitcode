package watcher

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type OnChangeFunc func()

type FileWatcher struct {
	dir       string
	interval  time.Duration
	onChange  OnChangeFunc
	stopCh    chan struct{}
	snapshots map[string]time.Time
	mu        sync.Mutex
}

func New(dir string, interval time.Duration, onChange OnChangeFunc) *FileWatcher {
	return &FileWatcher{
		dir:       dir,
		interval:  interval,
		onChange:  onChange,
		stopCh:    make(chan struct{}),
		snapshots: make(map[string]time.Time),
	}
}

func (w *FileWatcher) Start() {
	w.takeSnapshot()
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			if w.hasChanges() {
				log.Println("[WATCHER] changes detected, reloading...")
				w.onChange()
				w.takeSnapshot()
			}
		}
	}
}

func (w *FileWatcher) Stop() {
	close(w.stopCh)
}

func (w *FileWatcher) takeSnapshot() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.snapshots = make(map[string]time.Time)
	filepath.Walk(w.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".json" || filepath.Ext(path) == ".html" {
			w.snapshots[path] = info.ModTime()
		}
		return nil
	})
}

func (w *FileWatcher) hasChanges() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	changed := false
	filepath.Walk(w.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".json" && filepath.Ext(path) != ".html" {
			return nil
		}
		prev, exists := w.snapshots[path]
		if !exists || !info.ModTime().Equal(prev) {
			changed = true
		}
		return nil
	})

	return changed
}
