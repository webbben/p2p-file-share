package syncdir

import (
	"log"

	"github.com/fsnotify/fsnotify"
)

// start watching for file changes in the shared file directory, so changes can be communicated to other nodes
func WatchForFileChanges(dir string) {
	if dir == "" {
		log.Println("Failed to watch for file changes: no directory specified.")
		return
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	err = watcher.Add(dir)
	if err != nil {
		log.Fatal(err)
	}

	// watch for file events
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Println("Modified file:", event.Name, event.Op)
				// Handle file modification
			}
			if event.Op&fsnotify.Create == fsnotify.Create {
				log.Println("Created file:", event.Name, event.Op)
				// Handle file creation
			}
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				log.Println("Removed file:", event.Name)
				// Handle file removal
			}
		}
	}
}
