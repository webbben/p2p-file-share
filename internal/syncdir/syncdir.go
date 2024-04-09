package syncdir

import (
	"log"
	"strings"

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
			// ignore .swp files, which linux generates while editing some files
			if strings.HasSuffix(event.Name, ".swp") {
				continue
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Printf("Modified %s (%s)\n", event.Name, event.Op)
				// Handle file modification
			}
			if event.Op&fsnotify.Create == fsnotify.Create {
				log.Printf("Created %s (%s)\n", event.Name, event.Op)
				// Handle file creation
			}
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				log.Printf("Removed %s (%s)\n", event.Name, event.Op)
				// Handle file removal
			}
		}
	}
}
