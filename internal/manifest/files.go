package manifest

import (
	"os"
	"path/filepath"
	"strings"
)

var ignoredFiles = []string{
	".DS_Store", // macOS
	"Thumbs.db", // Windows
}

func isIgnoredFile(name string) bool {
	for _, ignored := range ignoredFiles {
		if strings.Contains(name, ignored) {
			return true
		}
	}
	return false
}

func FindManifestFiles(manifestDir string) ([]string, error) {
	var files []string
	err := filepath.Walk(manifestDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !isIgnoredFile(info.Name()) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}
