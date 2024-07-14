package util

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

func TestGetAllFilesUnderDirectory(t *testing.T) {
	wd := Getwd()
	if wd == "" {
		t.Error("failed to get working directory")
		return
	}
	testdir := filepath.Join(wd, "temp_test")

	err := SetEnvVars(map[string]string{
		"UTIL_ROOT":     wd,
		"UTIL_TEST_DIR": "temp_test",
	})
	if err != nil {
		t.Error("failed to set env vars:", err)
		return
	}

	if err = exec.Command("bash", filepath.Join(wd, "make_files_test1.sh")).Run(); err != nil {
		t.Error("failed to execute bash script:", err)
		return
	}
	defer os.RemoveAll(testdir)

	expFiles := []string{
		filepath.Join(testdir, "numbers.txt"),
		filepath.Join(testdir, "letters.txt"),
		filepath.Join(testdir, "sub1", "subfile.txt"),
		filepath.Join(testdir, "sub1", "anothersubfile.txt"),
	}

	files, err := GetAllFilesUnderDirectory(testdir)
	if err != nil {
		t.Error("failed to get files:", err)
		return
	}

	if len(files) != len(expFiles) {
		t.Errorf("incorrect number of files found. exp: %v, got: %v", expFiles, files)
		return
	}
	for _, file := range expFiles {
		if !slices.Contains(files, file) {
			t.Error("file missing:", file)
		}
	}
	log.Println("expected:", expFiles)
	log.Println("got:", files)
}
