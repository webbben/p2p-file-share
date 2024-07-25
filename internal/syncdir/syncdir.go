package syncdir

import (
	"crypto/md5"
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
	fileInfo, exists := fileIndex[filename]
	if !exists {
		log.Println("file is not indexed:", filename)
		return nil
	}
	return fileInfo
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
		fmt.Println("raw filename:", event.Name)
		fileChange := FileChange{
			File:     util.RemovePathPrefix(event.Name, dir),
			FullPath: event.Name,
		}

		// determine file change type
		if event.Op&fsnotify.Write == fsnotify.Write {
			log.Printf("Modified %s (%s)\n", event.Name, event.Op)
			fileChange.Change = FILE_MOD
		} else if event.Op&fsnotify.Create == fsnotify.Create {
			log.Printf("Created %s (%s)\n", event.Name, event.Op)
			fileChange.Change = FILE_MOD
		} else if event.Op&fsnotify.Remove == fsnotify.Remove {
			log.Printf("Removed %s (%s)\n", event.Name, event.Op)
			fileChange.Change = FILE_DEL
		} else {
			// ignore other changes types, such as CHMOD
			log.Printf("unhandled file change: %s (%s)\n", event.Name, event.Op)
			return nil, false
		}

		switch fileChange.Change {
		case FILE_MOD:
			isDir, err := util.IsDirectory(event.Name)
			if err != nil {
				log.Println("failed to get file info:", err)
			}
			waitForCompletion(event.Name, isDir)
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
				refreshIndex = true
				// restart filewatcher too, since we need to add new paths to the watcher
				return changes, true
			}
			return []FileChange{fileChange}, false
		case FILE_DEL:
			fileInfo := GetIndexedFileInfo(fileChange.File)
			if fileInfo != nil {
				fileChange.IsDir = fileInfo.IsDir()
			}
			refreshIndex = true
			return []FileChange{fileChange}, false
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

func waitForCompletion(fileName string, isDir bool) {
	var lastHash [16]byte
	log.Println("waiting for file completion...")
	for {
		var currentHash [16]byte
		if isDir {
			dirHash, err := hashDirectory(fileName)
			if err != nil {
				log.Println("error hashing directory:", err)
				return
			}
			currentHash = dirHash
		} else {
			fileHash, err := hashFile(fileName)
			if err != nil {
				log.Println("error hashing file:", err)
				return
			}
			currentHash = fileHash
		}
		if currentHash == lastHash {
			// No change detected in file content
			break
		}
		lastHash = currentHash
		time.Sleep(100 * time.Millisecond)
	}
	log.Println("file completed.")
}

func hashFile(fileName string) ([16]byte, error) {
	file, err := os.ReadFile(fileName)
	if err != nil {
		return [16]byte{}, err
	}
	return md5.Sum(file), nil
}

func hashDirectory(dirPath string) ([16]byte, error) {
	var combinedHash [16]byte

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileHash, err := hashFile(path)
			if err != nil {
				return err
			}
			combinedHash = combineHashes(combinedHash, fileHash)
		}
		return nil
	})

	if err != nil {
		return [16]byte{}, err
	}
	return combinedHash, nil
}

func combineHashes(hash1, hash2 [16]byte) [16]byte {
	var combined [16]byte
	for i := range hash1 {
		combined[i] = hash1[i] ^ hash2[i]
	}
	return combined
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
