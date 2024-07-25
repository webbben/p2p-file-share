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
	Exp        []FileChange
	Name       string
}

func TestAwaitNextFileChange(t *testing.T) {
	testCases := []ANFCTestCase{
		{
			ScriptName: "create.sh",
			Exp: []FileChange{
				{
					File:   "somefile.txt",
					IsDir:  false,
					Change: FILE_MOD,
				},
			},
			Name: "Create file",
		},
		{
			ScriptName: "modify.sh",
			Exp: []FileChange{{
				File:   "somefile.txt",
				IsDir:  false,
				Change: FILE_MOD,
			}},
			Name: "Modify file",
		},
		{
			ScriptName: "delete.sh",
			Exp: []FileChange{{
				File:   "somefile.txt",
				IsDir:  false,
				Change: FILE_DEL,
			}},
			Name: "Delete file",
		},
		{
			ScriptName: "sub_create.sh",
			Exp: []FileChange{{
				File:   "sub_directory/anotherfile.txt",
				IsDir:  false,
				Change: FILE_MOD,
			}},
			Name: "Create file in sub directory",
		},
		{
			ScriptName: "sub_modify.sh",
			Exp: []FileChange{{
				File:   "sub_directory/anotherfile.txt",
				IsDir:  false,
				Change: FILE_MOD,
			}},
			Name: "Modify file in sub directory",
		},
		{
			ScriptName: "sub_delete.sh",
			Exp: []FileChange{{
				File:   "sub_directory/anotherfile.txt",
				IsDir:  false,
				Change: FILE_DEL,
			}},
			Name: "Delete file in sub directory",
		},
		{
			ScriptName: "copy_dir.sh",
			Exp: []FileChange{
				{
					File:   "copydir/a.txt",
					IsDir:  false,
					Change: FILE_MOD,
				},
				{
					File:   "copydir/b.txt",
					IsDir:  false,
					Change: FILE_MOD,
				},
				{
					File:   "copydir/x/c.txt",
					IsDir:  false,
					Change: FILE_MOD,
				},
				{
					File:   "copydir/x/y/d.txt",
					IsDir:  false,
					Change: FILE_MOD,
				},
			},
			Name: "Copy directory",
		},
		{
			ScriptName: "delete_dir.sh",
			Exp: []FileChange{{
				File:   "copydir",
				IsDir:  true,
				Change: FILE_DEL,
			}},
			Name: "Delete directory",
		},
		{
			ScriptName: "copy_large_file.sh",
			Exp: []FileChange{{
				File:   "largefile",
				IsDir:  false,
				Change: FILE_MOD,
			}},
			Name: "Copy large file",
		},
		{
			ScriptName: "copy_large_dir.sh",
			Exp: []FileChange{
				{
					File:   "a/file_a",
					IsDir:  false,
					Change: FILE_MOD,
				},
				{
					File:   "a/file_aa",
					IsDir:  false,
					Change: FILE_MOD,
				},
				{
					File:   "a/b/file_b",
					IsDir:  false,
					Change: FILE_MOD,
				},
				{
					File:   "a/b/file_bb",
					IsDir:  false,
					Change: FILE_MOD,
				},
			},
			Name: "Copy large directory",
		},
	}

	wd := util.Getwd()
	if wd == "" {
		t.Error("failed to get working directory")
		return
	}
	testdir := filepath.Join(wd, "testwatch")
	// make sub_directory too since we will test detecting sub directory file actions
	if err := util.EnsureDir(filepath.Join(testdir, "sub_directory")); err != nil {
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
	watcherOpen := true
	defer watcher.Close()

	// env variables for the test scripts to use
	envVars := map[string]string{
		"SYNCDIR_WD":           wd,
		"SYNCDIR_ROOT":         testdir,
		"SYNCDIR_TEMP":         "test_temp",
		"SYNCDIR_SUB":          "sub_directory",
		"SYNCDIR_SUB_FILENAME": "anotherfile.txt",
		"SYNCDIR_FILENAME":     "somefile.txt",
		"SYNCDIR_COPYDIR":      "copydir",
	}
	if err = util.SetEnvVars(envVars); err != nil {
		t.Error("error setting env vars:", err)
		return
	}

	// some setup
	if err = exec.Command("bash", filepath.Join(wd, "test_scripts", "setup_copy_dir.sh")).Run(); err != nil {
		t.Error("error running setup script")
		return
	}
	if err = exec.Command("bash", filepath.Join(wd, "test_scripts", "setup_test_temp.sh")).Run(); err != nil {
		t.Error("error running temp directory setup script")
		return
	}
	defer os.RemoveAll(filepath.Join(wd, "test_temp"))

	// start tracking file changes
	RefreshFileIndex(testdir)
	detectedChanges := make([]FileChange, 0)
	go func() {
		restartCount := 0
		for {
			changes, restart := awaitNextFileChange(watcher, testdir)
			if restart {
				if !watcherOpen {
					break
				}
				restartCount++
				log.Println("watcher restart?")
				if restartCount > 3 {
					t.Error("too many watcher restarts; something bad must be happening...")
					return
				}
			}
			detectedChanges = changes
		}
	}()
	// wait a second to make sure the file change watcher is working
	time.Sleep(100 * time.Millisecond)

	for _, testCase := range testCases {
		log.Printf("%s: Begin\n", testCase.Name)
		if err = exec.Command("bash", filepath.Join(wd, "test_scripts", testCase.ScriptName)).Run(); err != nil {
			t.Error(testCase.Name+":", err)
			return
		}

		// wait up to 5-ish seconds for new changes to come in.
		// in reality it should only take at most a couple hundred milliseconds
		count := 0
		for len(detectedChanges) == 0 {
			time.Sleep(100 * time.Millisecond)
			count++
			if count > 50 {
				t.Log("waiting for changes too long...")
				break
			}
		}
		for _, expChange := range testCase.Exp {
			found := false
			for _, change := range detectedChanges {
				if expChange.IsSame(change) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("%s: missing file change: %v\n", testCase.Name, expChange)
			}
		}
		if len(testCase.Exp) != len(detectedChanges) {
			t.Errorf("%s: incorrect number of changes.\n", testCase.Name)
			t.Log("exp:", testCase.Exp)
			t.Log("got:", detectedChanges)
		}
		detectedChanges = make([]FileChange, 0)
		log.Printf("%s: End\n", testCase.Name)
	}
	watcherOpen = false
}
