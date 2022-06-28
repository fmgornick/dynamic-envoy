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
	Remove
)

type Message struct {
	Operation CMR
	Path      string
}

var watcher *fsnotify.Watcher

func Watch(directory string, change chan<- Message) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create new watcher: %+v", err)
	}
	defer watcher.Close()

	err = filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println(err)
			return nil
		}

		if info.IsDir() {
			fmt.Printf("monitering new directory: %s\n", path)
			return watcher.Add(path)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create sub-directory watchers: %+v", err)
	}

	done := make(chan bool)

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
				} else if event.Op&fsnotify.Rename == fsnotify.Rename {
					fmt.Printf("moved file: %s\n", event.Name)
					change <- Message{
						Operation: Move,
						Path:      event.Name,
					}
				} else if event.Op&fsnotify.Remove == fsnotify.Remove {
					fmt.Printf("removed file: %s\n", event.Name)
					change <- Message{
						Operation: Remove,
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
