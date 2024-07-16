package util

import (
	"log"
	"os"
	"path/filepath"
	"strings"
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

// remove a initial path prefix to get the relative path of a file
func RemovePathPrefix(fullPath string, prefixPath string) string {
	output := strings.TrimPrefix(fullPath, prefixPath)
	if output == "" {
		return ""
	}
	// remove a leading file path separator, so that the path string doesn't look like it starts from the root directory (/...)
	if rune(output[0]) == os.PathSeparator {
		output = output[1:]
	}
	return output
}

// indexes a directory, mapping each file or sub-directory to its FileInfo
func IndexDirectory(dir string) (map[string]os.FileInfo, error) {
	index := make(map[string]os.FileInfo)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		trimmedPath := RemovePathPrefix(path, dir)
		index[trimmedPath] = info
		return nil
	})
	return index, err
}
