package watcher

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

type CMR int

const (
	Create CMR = iota
	Modify
	Move
	Delete
)

type Message struct {
	Operation CMR
	Path      string
}

var watcher *fsnotify.Watcher

// keeps track of changes in specified directory and any sub-directories
func Watch(directory string, change chan<- Message) error {
	// initialize watcher
	var err error
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create new watcher: %+v", err)
	}
	defer watcher.Close()

	// add watchers to initial directory tree
	if err = filepath.Walk(directory, addWatchers); err != nil {
		return err
	}

	done := make(chan bool)

	// wait for changes and notify main
	// also add new watchers if a directory is added
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					fmt.Printf("modified file: %s\n", event.Name)
					change <- Message{
						Operation: Modify,
						Path:      event.Name,
					}
				} else if event.Op&fsnotify.Create == fsnotify.Create {
					fmt.Printf("added file: %s\n", event.Name)
					change <- Message{
						Operation: Create,
						Path:      event.Name,
					}
					fileInfo, _ := os.Stat(event.Name)
					if fileInfo.IsDir() {
						// add new directory watcher
						if err = filepath.Walk(event.Name, addWatchers); err != nil {
							fmt.Printf("%+v\n", err)
						}
					}
				} else if event.Op&fsnotify.Rename == fsnotify.Rename {
					fmt.Printf("moved file: %s\n", event.Name)
					change <- Message{
						Operation: Move,
						Path:      event.Name,
					}
				} else if event.Op&fsnotify.Remove == fsnotify.Remove {
					fmt.Printf("deleted file: %s\n", event.Name)
					change <- Message{
						Operation: Delete,
						Path:      event.Name,
					}
				}

			case err := <-watcher.Errors:
				fmt.Printf("watcher error: %+v\n", err)
			}
		}
	}()

	<-done
	return nil
}

// walk function to add watcher to any directory stemming from root (inclusive)
func addWatchers(root string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if info.IsDir() {
		err := watcher.Add(root)
		if err != nil {
			return fmt.Errorf("failed to add watcher to %s: %+v", root, err)
		}
		fmt.Printf("monitering new directory: %s\n", root)
	}
	return nil
}
