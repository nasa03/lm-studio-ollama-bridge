package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func logInfo(message string, args ...interface{}) {
	fmt.Printf("\033[1;34m[INFO]\033[0m "+message+"\n", args...)
}

func logError(message string, args ...interface{}) {
	fmt.Printf("\033[1;31m[ERROR]\033[0m "+message+"\n", args...)
}

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logError("Failed to get user home directory: %v", err)
		return
	}

	manifestDir := filepath.Join(homeDir, ".ollama", "models", "manifests", "registry.ollama.ai")
	blobDir := filepath.Join(homeDir, ".ollama", "models", "blobs")
	publicModelsDir := filepath.Join(homeDir, ".cache", "lm-studio", "models")
	lmstudioDir := filepath.Join(publicModelsDir, "ollama")

	logInfo("Manifest Directory: %s", manifestDir)
	logInfo("Blob Directory: %s", blobDir)
	logInfo("Public Models Directory: %s", publicModelsDir)

	var manifestFiles []string
	err = filepath.Walk(manifestDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !strings.HasSuffix(info.Name(), ".DS_Store") {
			manifestFiles = append(manifestFiles, path)
		}
		return nil
	})
	if err != nil {
		logError("Error walking through manifest directory: %v", err)
		return
	}

	for _, file := range manifestFiles {
		logInfo("Exploring manifest directory: %s", file)
	}

	for _, manifestPath := range manifestFiles {
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			logError("Failed to read manifest file %s: %v", manifestPath, err)
			continue
		}

		var manifest Manifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			logError("Failed to parse JSON in file %s: %v", manifestPath, err)
			continue
		}

		configDigest := strings.Replace(manifest.Config.Digest, "sha256:", "sha256-", 1)
		modelConfigPath := filepath.Join(blobDir, configDigest)

		var modelFile string
		for _, layer := range manifest.Layers {
			if strings.HasSuffix(layer.MediaType, "model") {
				digest := strings.Replace(layer.Digest, "sha256:", "sha256-", 1)
				modelFile = filepath.Join(blobDir, digest)
				break
			}
		}
		if modelFile == "" {
			logError("No valid model file found in %s, skipping", manifestPath)
			continue
		}

		modelConfigData, err := os.ReadFile(modelConfigPath)
		if err != nil {
			logError("Failed to read model config file %s: %v", modelConfigPath, err)
			continue
		}

		var modelConfig ModelConfig
		if err := json.Unmarshal(modelConfigData, &modelConfig); err != nil {
			logError("Failed to parse model config JSON in %s: %v", modelConfigPath, err)
			continue
		}

		modelName := filepath.Base(filepath.Dir(manifestPath))
		modelDir := filepath.Join(lmstudioDir, modelName)

		logInfo("Model name: %s", modelName)
		logInfo("Quant is: %s", modelConfig.ModelType)
		logInfo("Number of parameters trained on: %s", modelConfig.ModelType)
		logInfo("Model format: %s", modelConfig.ModelFormat)
		logInfo("Model path: %s", modelFile)

		if err := os.MkdirAll(modelDir, 0755); err != nil {
			logError("Failed to create model directory %s: %v", modelDir, err)
			continue
		}

		symlinkName := filepath.Join(modelDir, fmt.Sprintf("%s-%s-%s.%s", modelName, modelConfig.ModelType, modelConfig.FileType, modelConfig.ModelFormat))
		if _, err := os.Lstat(symlinkName); err == nil {
			os.Remove(symlinkName)
		}
		if err := os.Symlink(modelFile, symlinkName); err != nil {
			logError("Failed to create symbolic link for %s: %v", modelFile, err)
			continue
		}
		logInfo("Created symbolic link for %s -> %s", modelFile, symlinkName)
	}

	logInfo("lm-studio-ollama-bridge complete.")
	logInfo("Models have been linked to  %s", lmstudioDir)
}
