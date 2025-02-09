package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

type Manifest struct {
	Config struct {
		Digest string `json:"digest"`
	} `json:"config"`
	Layers []struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
	} `json:"layers"`
}

type ModelConfig struct {
	FileType    string `json:"file_type"`
	ModelFormat string `json:"model_format"`
	ModelType   string `json:"model_type"`
}

func initLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		ForceColors:     true,
		DisableSorting:  false,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	// logger.SetReportCaller(true)
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)
	return logger
}

func findManifestFiles(manifestDir string) ([]string, error) {
	var manifestFiles []string
	err := filepath.Walk(manifestDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !strings.HasSuffix(info.Name(), ".DS_Store") {
			manifestFiles = append(manifestFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return manifestFiles, nil
}

func normalizeDigest(digest string) string {
	return strings.Replace(digest, "sha256:", "sha256-", 1)
}

// processManifest reads and processes a single manifest file, then creates the appropriate symlink.
func processManifest(manifestPath, blobDir, lmstudioDir string, logger *logrus.Logger) error {
	logger.WithField("manifest_path", manifestPath).Info("Starting processing of manifest")
	
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest file: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("failed to unmarshal manifest JSON: %w", err)
	}

	configDigest := normalizeDigest(manifest.Config.Digest)
	modelConfigPath := filepath.Join(blobDir, configDigest)

	var modelFile string
	for _, layer := range manifest.Layers {
		if strings.HasSuffix(layer.MediaType, "model") {
			digest := normalizeDigest(layer.Digest)
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
	modelDir := filepath.Join(lmstudioDir, modelName)
	logger.WithFields(logrus.Fields{
		"model_name":   modelName,
		"model_type":   modelConfig.ModelType,
		"model_format": modelConfig.ModelFormat,
		"model_file":   modelFile,
	}).Info("Extracted model details")

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

	if err := os.Symlink(modelFile, symlinkName); err != nil {
		return fmt.Errorf("failed to create symbolic link from %s to %s: %w", modelFile, symlinkName, err)
	}

	logger.WithFields(logrus.Fields{
		"model_file": modelFile,
		"symlink":    symlinkName,
	}).Info("Successfully created symbolic link")

	return nil
}

func main() {
	logger := initLogger()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.WithError(err).Error("Failed to get user home directory")
		return
	}

	manifestDir := filepath.Join(homeDir, ".ollama", "models", "manifests", "registry.ollama.ai")
	blobDir := filepath.Join(homeDir, ".ollama", "models", "blobs")
	publicModelsDir := filepath.Join(homeDir, ".cache", "lm-studio", "models")
	lmstudioDir := filepath.Join(publicModelsDir, "ollama")

	logger.WithFields(logrus.Fields{
		"manifest_dir": manifestDir,
		"blob_dir":     blobDir,
		"public_dir":   publicModelsDir,
	}).Info("Application directories determined")

	manifestFiles, err := findManifestFiles(manifestDir)
	if err != nil {
		logger.WithError(err).Error("Error while searching for manifest files")
		return
	}
	if len(manifestFiles) == 0 {
		logger.Warn("No manifest files found")
		return
	}

	for _, manifestPath := range manifestFiles {
		logger.WithField("manifest_path", manifestPath).Info("Found manifest file")
		if err := processManifest(manifestPath, blobDir, lmstudioDir, logger); err != nil {
			logger.WithError(err).WithField("manifest_path", manifestPath).Error("Failed to process manifest")
		}
	}

	logger.Info("lm-studio-ollama-bridge complete.")
	logger.Infof("Models have been linked to %s", lmstudioDir)
}
