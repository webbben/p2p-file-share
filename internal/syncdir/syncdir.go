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
	"github.com/webbben/p2p-file-share/internal/util"
)

type FileChange struct {
	File     string
	FullPath string // full path of the file that was changed
	Change   string
	IsDir    bool
}

func (fc FileChange) String() string {
	return fmt.Sprintf("%s (%s)", fc.File, fc.Change)
}

func (fc FileChange) IsSame(c FileChange) bool {
	return fc.File == c.File && fc.Change == c.Change && fc.IsDir == c.IsDir
}

const (
	FILE_MOD string = "mod" // file modified (or created) - signals a file should be copied over to other nodes
	FILE_DEL string = "del" // file deleted - signals a file should be deleted from other nodes
)

var (
	changeFlag            bool         = false          // flag for when changes have been detected
	changedFiles          []FileChange = []FileChange{} // file changes queued up to be broadcast
	applyingRemoteChanges bool         = false          // flag for when remote changes are being applied

	fileIndex map[string]os.FileInfo // index of all files and their info
)

func GetIndexedFileInfo(filename string) os.FileInfo {
	return fileIndex[filename]
}

func RefreshFileIndex(dir string) {
	index, err := util.IndexDirectory(dir)
	if err != nil {
		log.Printf("failed to index %s: %s", dir, err)
		return
	}
	fileIndex = index
}

// start watching for file changes in the shared file directory, so changes can be communicated to other nodes
func WatchForFileChanges(dir string) {
	if dir == "" {
		log.Println("Failed to watch for file changes: no directory specified.")
		return
	}
	watcher, err := getWatcher(dir)
	if err != nil {
		log.Fatal("failed to set up file watcher:", err)
	}
	defer watcher.Close()

	RefreshFileIndex(dir)

	// watch for file events
	for {
		fileChanges, restart := awaitNextFileChange(watcher, dir)
		for _, change := range fileChanges {
			queueFileChange(change)
		}
		if restart {
			break
		}
	}
	// if an error occurs with fsnotify, try waiting a few seconds before restarting
	fmt.Println("Restarting file watcher in a second...")
	time.Sleep(1 * time.Second)
	go WatchForFileChanges(dir)
}

// makes a watcher that watches for file changes in the given directory and all sub-directories
func getWatcher(dir string) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	err = watcher.Add(dir)
	if err != nil {
		return nil, err
	}

	// recursively add all sub-directories to the watcher too
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			err = watcher.Add(path)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return watcher, nil
}

// return the next filechange
//
// returns: FileChange, whether to restart filewatcher (e.g. if an error happens and we want to restart)
func awaitNextFileChange(watcher *fsnotify.Watcher, dir string) ([]FileChange, bool) {
	refreshIndex := false
	defer func() {
		if refreshIndex {
			RefreshFileIndex(dir)
		}
	}()

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
		if ignoreFile(event.Name) {
			return nil, false
		}

		fmt.Println("raw filename:", event.Name, "base dir:", dir)
		fileChange := FileChange{
			File:     util.RemovePathPrefix(event.Name, dir),
			FullPath: event.Name,
		}
		if event.Op&fsnotify.Write == fsnotify.Write {
			// file modified
			log.Printf("Modified %s (%s)\n", event.Name, event.Op)
			fileChange.Change = FILE_MOD
			return []FileChange{fileChange}, false
		} else if event.Op&fsnotify.Create == fsnotify.Create {
			refreshIndex = true
			// file created
			log.Printf("Created %s (%s)\n", event.Name, event.Op)
			// if its a new directory, add any files found under it
			isDir, err := util.IsDirectory(event.Name)
			if err != nil {
				log.Println("failed to get file info:", err)
			}
			fileChange.IsDir = isDir
			if isDir {
				changes := make([]FileChange, 0)
				nestedFiles, err := util.GetAllFilesUnderDirectory(fileChange.FullPath)
				if err != nil {
					log.Println("failed to get nested files:", err)
					return nil, false
				}
				log.Println("files added from new directory:", nestedFiles)
				for _, file := range nestedFiles {
					changes = append(changes, FileChange{
						File:   util.RemovePathPrefix(file, dir),
						Change: FILE_MOD,
					})
				}
				// restart filewatcher too, since we need to add new paths to the watcher
				return changes, true
			}
			fileChange.Change = FILE_MOD
			return []FileChange{fileChange}, false
		} else if event.Op&fsnotify.Remove == fsnotify.Remove {
			refreshIndex = true
			// file removed
			log.Printf("Removed %s (%s)\n", event.Name, event.Op)
			// get file info from previous index, since the file no longer exists
			fileInfo := GetIndexedFileInfo(fileChange.File)
			if fileInfo.IsDir() {
				fileChange.IsDir = true
			}
			fileChange.Change = FILE_DEL
			return []FileChange{fileChange}, false
		} else {
			// other change?
			log.Printf("unhandled file change: %s (%s)\n", event.Name, event.Op)
		}
	case err, ok := <-watcher.Errors:
		if !ok {
			log.Println("WARNING! file watcher encountered an error:", err)
			return nil, true
		}
		log.Println("File watcher error:", err)
	}
	return nil, false
}

// returns whether the file should be ignored or not, such as if it's some autogenerated file for specific OS
func ignoreFile(filename string) bool {
	// ignore .swp files, which linux generates while editing some files
	if strings.HasSuffix(filename, ".swp") {
		return true
	}
	// ignore .DS_Store files, which is just metadata for directories in macOS
	if strings.HasSuffix(filename, ".DS_Store") {
		return true
	}
	return false
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
			IsDir:  fileChange.IsDir,
			Change: fileChange.Change,
		})
	}
}

// handle a file change notification sent to this node from a peer
func HandleRemoteFileChange(fileChange m.NotifyFileChange, remoteIP string, config c.Config) {
	if fileChange.File == "" {
		log.Println("error handling remote file change: no file name provided")
		return
	}
	if fileChange.Change == "" {
		log.Println("error handling remote file change: no file change type provided (needs mod, del, etc)")
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
			log.Println("error requesting file change:", err)
			return
		}
		fmt.Println("successfully retrieved file change from peer:", fileChange.File)
	case FILE_DEL:
		// delete the file
		fmt.Println("received file deletion change")
		filePath := getFullFilePath(fileChange.File, config)
		if filePath == "" {
			log.Println("failed to delete file; no filepath provided")
			return
		}
		if fileChange.IsDir {
			if err := os.RemoveAll(filePath); err != nil {
				log.Println("failed to remove directory:", err)
			}
			return
		}
		if err := os.Remove(filePath); err != nil {
			log.Println("failed to remove file:", err)
		}
	}
}
