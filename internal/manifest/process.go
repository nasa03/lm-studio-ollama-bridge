package manifest

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

// ProcessManifest reads and processes a single manifest file and creates symlinks in all destination directories.
func ProcessManifest(manifestPath, blobDir string, destDirs []string, logger *logrus.Logger) error {
	logger.WithField("manifest_path", manifestPath).Info("Starting processing of manifest")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest file: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("failed to unmarshal manifest JSON: %w", err)
	}

	configDigest := NormalizeDigest(m.Config.Digest)
	modelConfigPath := filepath.Join(blobDir, configDigest)

	var modelFile string
	for _, layer := range m.Layers {
		if strings.HasSuffix(layer.MediaType, "model") {
			digest := NormalizeDigest(layer.Digest)
			modelFile = filepath.Join(blobDir, digest)
			break
		}
	}
	if modelFile == "" {
		return fmt.Errorf("no valid model file found in manifest")
	}

	modelConfigData, err := os.ReadFile(modelConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read model config file: %w", err)
	}

	var modelConfig ModelConfig
	if err := json.Unmarshal(modelConfigData, &modelConfig); err != nil {
		return fmt.Errorf("failed to unmarshal model config JSON: %w", err)
	}

	modelName := filepath.Base(filepath.Dir(manifestPath))

	for _, destDir := range destDirs {
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create destination directory %s: %w", destDir, err)
		}

		modelDir := filepath.Join(destDir, modelName)
		if err := os.MkdirAll(modelDir, 0755); err != nil {
			return fmt.Errorf("failed to create model directory %s: %w", modelDir, err)
		}

		symlinkName := filepath.Join(modelDir,
			fmt.Sprintf("%s-%s-%s.%s", modelName, modelConfig.ModelType, modelConfig.FileType, modelConfig.ModelFormat))

		if _, err := os.Lstat(symlinkName); err == nil {
			if err := os.Remove(symlinkName); err != nil {
				logger.WithField("symlink", symlinkName).Warn("Failed to remove existing symlink")
			} else {
				logger.WithField("symlink", symlinkName).Info("Removed existing symlink")
			}
		}

		// Create symlink with a Windows fallback.
		if runtime.GOOS == "windows" {
			if err := os.Symlink(modelFile, symlinkName); err != nil {
				logger.WithError(err).Warn("Symlink creation failed on Windows; attempting file copy as fallback")
				if copyErr := copyFile(modelFile, symlinkName); copyErr != nil {
					return fmt.Errorf("failed to create file copy as fallback: %w", copyErr)
				}
			}
		} else {
			if err := os.Symlink(modelFile, symlinkName); err != nil {
				return fmt.Errorf("failed to create symbolic link from %s to %s: %w", modelFile, symlinkName, err)
			}
		}

		logger.WithFields(logrus.Fields{
			"model_file": modelFile,
			"symlink":    symlinkName,
		}).Info("Successfully created symbolic link (or file copy fallback if not supported on Windows)")
	}

	return nil
}

func copyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
