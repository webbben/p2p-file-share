package syncdir

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/webbben/p2p-file-share/internal/util"
)

type ANFCTestCase struct {
	ScriptName string
	Exp        FileChange
	Name       string
}

func TestAwaitNextFileChange(t *testing.T) {
	testCases := []ANFCTestCase{
		{
			ScriptName: "create.sh",
			Exp: FileChange{
				File:   "somefile.txt",
				Change: FILE_MOD,
			},
			Name: "Create file",
		},
		{
			ScriptName: "modify.sh",
			Exp: FileChange{
				File:   "somefile.txt",
				Change: FILE_MOD,
			},
			Name: "Modify file",
		},
		{
			ScriptName: "delete.sh",
			Exp: FileChange{
				File:   "somefile.txt",
				Change: FILE_DEL,
			},
			Name: "Delete file",
		},
		{
			ScriptName: "sub_create.sh",
			Exp: FileChange{
				File:   "sub_directory/anotherfile.txt",
				Change: FILE_MOD,
			},
			Name: "Create file in sub directory",
		},
		{
			ScriptName: "sub_modify.sh",
			Exp: FileChange{
				File:   "sub_directory/anotherfile.txt",
				Change: FILE_MOD,
			},
			Name: "Modify file in sub directory",
		},
		{
			ScriptName: "sub_delete.sh",
			Exp: FileChange{
				File:   "sub_directory/anotherfile.txt",
				Change: FILE_DEL,
			},
			Name: "Modify file in sub directory",
		},
	}

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

	watcher, err := getWatcher(testdir)
	if err != nil {
		t.Error(err)
		return
	}
	defer watcher.Close()

	// env variables for the test scripts to use
	envVars := map[string]string{
		"SYNCDIR_ROOT":         testdir,
		"SYNCDIR_SUB":          "sub_directory",
		"SYNCDIR_SUB_FILENAME": "anotherfile.txt",
		"SYNCDIR_FILENAME":     "somefile.txt",
	}
	if err = util.SetEnvVars(envVars); err != nil {
		t.Error("error setting env vars:", err)
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
	// wait a second to make sure the file change watcher is working
	time.Sleep(100 * time.Millisecond)

	for _, testCase := range testCases {
		if err = exec.Command("bash", filepath.Join(wd, testCase.ScriptName)).Run(); err != nil {
			t.Error(testCase.Name+":", err)
			return
		}
		time.Sleep(100 * time.Millisecond)
		if detectedChange.File != testCase.Exp.File || detectedChange.Change != testCase.Exp.Change {
			t.Errorf("%s: wrong file change detected. expected: %q, got: %q", testCase.Name, testCase.Exp, detectedChange)
		}
		detectedChange = FileChange{}
	}
}
