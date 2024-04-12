package syncdir

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	c "github.com/webbben/p2p-file-share/internal/config"
	filetransfer "github.com/webbben/p2p-file-share/internal/file-transfer"
	messagebroker "github.com/webbben/p2p-file-share/internal/message-broker"
	m "github.com/webbben/p2p-file-share/internal/model"
)

type FileChange struct {
	File   string
	Change string
}

const (
	FILE_MOD string = "mod" // file modified (or created) - signals a file should be copied over to other nodes
	FILE_DEL string = "del" // file deleted - signals a file should be deleted from other nodes
)

var (
	changeFlag            bool         = false          // flag for when changes have been detected
	changedFiles          []FileChange = []FileChange{} // file changes queued up to be broadcast
	applyingRemoteChanges bool         = false          // flag for when remote changes are being applied
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
			// ignore if these changes are just being transferred from other nodes
			if applyingRemoteChanges {
				fmt.Println("remote change: ignore file event")
				return
			}
			// ignore .swp files, which linux generates while editing some files
			if strings.HasSuffix(event.Name, ".swp") {
				continue
			}
			fmt.Println("raw filename:", event.Name, "base dir:", dir)
			filename := removePathPrefix(event.Name, dir)
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Printf("Modified %s (%s)\n", event.Name, event.Op)
				// Handle file modification
				queueFileChange(filename, FILE_MOD)
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

// remove a initial path prefix to get the relative path of a file
func removePathPrefix(fullPath string, prefixPath string) string {
	output := strings.TrimPrefix(fullPath, prefixPath)
	// remove a leading file path separator, so that the path string doesn't look like it starts from the root directory (/...)
	if rune(output[0]) == os.PathSeparator {
		output = output[1:]
	}
	return output
}

// queues up a file change and triggers the file changes broadcast after a short delay
func queueFileChange(file string, changeType string) {
	file = strings.TrimSpace(file)
	if file == "" {
		fmt.Println("failed to queue change: empty file name!")
		return
	}
	// ship the file changes after a short delay, to make sure any simultaneous file changes are all accounted for together
	// sometimes when a file is changed, under the hood there are multiple changes occurring (a WRITE and CHMOD, for example), and we don't want to send out multiple file change notifications in such cases.
	if !changeFlag {
		go func() {
			time.Sleep(1 * time.Second)
			shipFileChanges()
		}()
		changeFlag = true
	}
	for _, f := range changedFiles {
		if f.File == file {
			return
		}
	}
	changedFiles = append(changedFiles, FileChange{File: file, Change: changeType})
}

// ships file changes to be broadcast to other nodes.
func shipFileChanges() {
	if len(changedFiles) == 0 {
		fmt.Println("no file changes queued?")
		return
	}
	// reset changes
	defer func() {
		changedFiles = []FileChange{}
		changeFlag = false
	}()
	// broadcast file changes
	for _, fileChange := range changedFiles {
		messagebroker.BroadcastMessage(m.NotifyFileChange{
			Type:   c.TYPE_FILE_CHANGE_NOTIFY,
			File:   fileChange.File,
			Change: fileChange.Change,
		})
	}
}

// handle a file change notification sent to this node from a peer
func HandleRemoteFileChange(fileChange m.NotifyFileChange, remoteIP string) {
	if fileChange.File == "" {
		fmt.Println("error handling remote file change: no file name provided")
		return
	}
	if fileChange.Change == "" {
		fmt.Println("error handling remote file change: no file change type provided (needs mod, del, etc)")
		return
	}
	// flag that incoming changes are remote - and shouldn't be rebroadcasted
	applyingRemoteChanges = true
	defer func() {
		applyingRemoteChanges = false
	}()

	switch fileChange.Change {
	case FILE_MOD:
		_, err := filetransfer.RequestFile(remoteIP, fileChange.File)
		if err != nil {
			fmt.Println("error requesting file change:", err)
			return
		}
		fmt.Println("successfully retrieved file change from peer:", fileChange.File)
	case FILE_DEL:
		// delete the file
		fmt.Println("received file deletion change")
	}
}
