package syncdir

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/webbben/p2p-file-share/internal/util"
)

func TestAwaitNextFileChange(t *testing.T) {
	wd := util.Getwd()
	if wd == "" {
		t.Error("failed to get working directory")
		return
	}
	testdir := filepath.Join(wd, "testwatch")
	if err := util.EnsureDir(testdir); err != nil {
		t.Error("failed to create test directory:", err)
		return
	}
	// delete the testwatch directory after the tests are done
	defer os.RemoveAll(testdir)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Error(err)
		return
	}
	defer watcher.Close()

	err = watcher.Add(testdir)
	if err != nil {
		t.Error(err)
		return
	}

	testfile := filepath.Join(testdir, "somefile.txt")
	if err = os.Setenv("SYNCDIR_FILENAME", testfile); err != nil {
		t.Error("error setting env var:", err)
		return
	}

	// start tracking file changes
	detectedChange := FileChange{}
	go func() {
		for {
			change, restart := awaitNextFileChange(watcher, testdir)
			if restart {
				log.Println("watcher restart?")
				return
			}
			detectedChange = *change
		}
	}()

	// create file
	if err = exec.Command("bash", filepath.Join(wd, "create.sh")).Run(); err != nil {
		t.Error("failed to create file:", err)
		return
	}
	exp := FileChange{File: "somefile.txt", Change: FILE_MOD}
	time.Sleep(100 * time.Millisecond)
	if detectedChange.File != exp.File || detectedChange.Change != exp.Change {
		t.Errorf("Create file: wrong file change detected. expected: %q, got: %q", exp, detectedChange)
		return
	}
	detectedChange = FileChange{}

	// change file
	if err = exec.Command("bash", filepath.Join(wd, "modify.sh")).Run(); err != nil {
		t.Error("failed to modify file:", err)
		return
	}
	exp = FileChange{File: "somefile.txt", Change: FILE_MOD}
	time.Sleep(100 * time.Millisecond)
	if detectedChange.File != exp.File || detectedChange.Change != exp.Change {
		t.Errorf("Modify file: wrong file change detected. expected: %q, got: %q", exp, detectedChange)
		return
	}
	detectedChange = FileChange{}

	// delete file
	if err = exec.Command("bash", filepath.Join(wd, "delete.sh")).Run(); err != nil {
		t.Error("failed to delete file:", err)
		return
	}
	exp = FileChange{File: "somefile.txt", Change: FILE_DEL}
	time.Sleep(100 * time.Millisecond)
	if detectedChange.File != exp.File || detectedChange.Change != exp.Change {
		t.Errorf("Delete file: wrong file change detected. expected: %q, got: %q", exp, detectedChange)
		return
	}
}
