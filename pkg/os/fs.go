package os

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Exists function checks if the file/directory exists.
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// RemoveDir removes the directory.
func RemoveDir(path string) error {
	path, err := filterPath(path)
	if err != nil {
		return err
	}
	log.Println("Remove directory " + path)
	err = os.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("removeDir -> cannot remove: %w; dir=%s", err, path)
	}
	return nil
}

// RecreateDir removes the directory and creates it again.
func RecreateDir(path string) error {
	path, err := filterPath(path)
	if err != nil {
		return err
	}
	err = RemoveDir(path)
	if err != nil {
		return err
	}
	log.Println("Create directory " + path)
	err = os.Mkdir(path, 0755)
	if err != nil {
		return fmt.Errorf("recreateDir -> cannot create directory: %w; dir=%s", err, path)
	}
	return nil
}

func filterPath(path string) (string, error) {
	path, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		return "", fmt.Errorf("filterDir -> invalid path: %w; dir=%s", err, path)
	}
	if path == "/" {
		return "", fmt.Errorf("filterDir -> are you kidding me")
	}
	return path, nil
}
