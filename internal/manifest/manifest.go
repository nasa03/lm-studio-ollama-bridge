package manifest

import "strings"

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

func NormalizeDigest(digest string) string {
	return strings.Replace(digest, "sha256:", "sha256-", 1)
}
