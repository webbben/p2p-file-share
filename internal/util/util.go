package util

import (
	"log"
	"os"
	"path/filepath"
)

// gets the current working directory of the project code. mainly used for debugging, unit tests, etc.
func Getwd() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Println("failed to get working directory:", err)
		return ""
	}
	absPath, err := filepath.Abs(wd)
	if err != nil {
		log.Println("error getting absolute path:", err)
		return ""
	}
	return absPath
}

// makes sure a directory exists, and if not it creates it.
// makes directories recursively if needed (multiple nested dirs that don't exist yet).
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, os.ModePerm)
}

// sets a bunch of environment variables from the given map.
func SetEnvVars(varMap map[string]string) error {
	for k, v := range varMap {
		if err := os.Setenv(k, v); err != nil {
			return err
		}
	}
	return nil
}

func IsDirectory(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

// Gets all files under a directory. returns a list of their absolute paths. ignores directories.
func GetAllFilesUnderDirectory(dir string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
