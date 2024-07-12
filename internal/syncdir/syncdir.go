package syncdir

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
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
		fileChange, restart := awaitNextFileChange(watcher, dir)
		if restart {
			break
		}
		if fileChange == nil {
			continue
		}
		queueFileChange(*fileChange)
	}
	// if an error occurs with fsnotify, try waiting a few seconds before restarting
	fmt.Println("Restarting file watcher in a few seconds...")
	time.Sleep(5 * time.Second)
	go WatchForFileChanges(dir)
}

// return the next filechange
//
// returns: FileChange, whether to restart filewatcher (e.g. if an error happens and we want to restart)
func awaitNextFileChange(watcher *fsnotify.Watcher, dir string) (*FileChange, bool) {
	select {
	case event, ok := <-watcher.Events:
		if !ok {
			log.Println("WARNING: file change watcher closed unexpectedly!")
			return nil, true
		}
		// ignore if these changes are just being transferred from other nodes
		if applyingRemoteChanges {
			fmt.Println("remote change: ignore file event")
			return nil, false
		}
		// ignore .swp files, which linux generates while editing some files
		if strings.HasSuffix(event.Name, ".swp") {
			return nil, false
		}
		fmt.Println("raw filename:", event.Name, "base dir:", dir)
		filename := removePathPrefix(event.Name, dir)
		if event.Op&fsnotify.Write == fsnotify.Write {
			// file modified
			log.Printf("Modified %s (%s)\n", event.Name, event.Op)
			return &FileChange{File: filename, Change: FILE_MOD}, false
		} else if event.Op&fsnotify.Create == fsnotify.Create {
			// file created
			log.Printf("Created %s (%s)\n", event.Name, event.Op)
			return &FileChange{File: filename, Change: FILE_MOD}, false
		} else if event.Op&fsnotify.Remove == fsnotify.Remove {
			// file removed
			log.Printf("Removed %s (%s)\n", event.Name, event.Op)
			return &FileChange{File: filename, Change: FILE_DEL}, false
		} else {
			// other change?
			log.Printf("unhandled file change: %s (%s)\n", event.Name, event.Op)
		}
	case err, ok := <-watcher.Errors:
		if !ok {
			log.Println("WARNING: file watcher encountered an error!")
			return nil, true
		}
		log.Println("File watcher error:", err)
	}
	return nil, false
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

func getFullFilePath(filename string, config c.Config) string {
	if config.SharedDirectoryPath == "" {
		fmt.Println("failed to get full file path: missing config")
		return ""
	}
	return filepath.Join(config.SharedDirectoryPath, filename)
}

// queues up a file change and triggers the file changes broadcast after a short delay
func queueFileChange(fileChange FileChange) {
	fileChange.File = strings.TrimSpace(fileChange.File)
	if fileChange.File == "" {
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
	// make sure the given filechange isn't already queued
	for _, f := range changedFiles {
		if f.File == fileChange.File {
			if f.Change != fileChange.Change {
				log.Printf("file (%s) already queued, but has a different change type? (prev: %s, new: %s)\n", f.File, f.Change, fileChange.Change)
			}
			return
		}
	}
	changedFiles = append(changedFiles, fileChange)
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
func HandleRemoteFileChange(fileChange m.NotifyFileChange, remoteIP string, config c.Config) {
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
		filePath := getFullFilePath(fileChange.File, config)
		if filePath == "" {
			fmt.Println("failed to delete file; no filepath provided")
			return
		}
		if err := os.Remove(filePath); err != nil {
			fmt.Println("failed to remove file:", err)
		}
	}
}
