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
