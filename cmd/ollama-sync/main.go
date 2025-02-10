package main

import (
	"fmt"
	"runtime"

	"github.com/ishan-marikar/lm-studio-ollama-bridge/internal/config"
	"github.com/ishan-marikar/lm-studio-ollama-bridge/internal/logger"
	"github.com/ishan-marikar/lm-studio-ollama-bridge/internal/manifest"
	"github.com/sirupsen/logrus"
)




func displayWelcomeMessage() {
    asciiArt := `

        '||  '||                                                               
  ...    ||   ||   ....   .. .. ..    ....      ....  .... ... .. ...     ....  
.|  '|.  ||   ||  '' .||   || || ||  '' .||    ||. '   '|.  |   ||  ||  .|   '' 
||   ||  ||   ||  .|' ||   || || ||  .|' ||    . '|..   '|.|    ||  ||  ||      
 '|..|' .||. .||. '|..'|' .|| || ||. '|..'|'   |'..|'    '|    .||. ||.  '|...' 
                                                      .. |                      
                                                       ''                       

    `
    appName := "Ollama Sync"
    version := "1.0.0"
    description := "A tool to bridge Ollama models with other tools (like LM Studio) seamlessly."

    fmt.Println(asciiArt)
    fmt.Printf("%s - Version %s\n", appName, version)
    fmt.Println(description)
    fmt.Println()
}


func main() {

	displayWelcomeMessage()

	log := logger.InitLogger()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.WithError(err).Error("Failed to load configuration")
		return
	}

	log.Infof("Running on %s", runtime.GOOS)

	log.WithFields(logrus.Fields{
		"manifest_dir":  cfg.ManifestDir,
		"blob_dir":      cfg.BlobDir,
		"destinations":  cfg.Destinations,
	}).Info("Application directories determined from config")

	manifestFiles, err := manifest.FindManifestFiles(cfg.ManifestDir)
	if err != nil {
		log.WithError(err).Error("Error while searching for manifest files")
		return
	}
	if len(manifestFiles) == 0 {
		log.Warn("No manifest files found")
		return
	}

	for _, filePath := range manifestFiles {
		log.WithField("manifest_path", filePath).Info("Found manifest file")
		if err := manifest.ProcessManifest(filePath, cfg.BlobDir, cfg.Destinations, log); err != nil {
			log.WithError(err).WithField("manifest_path", filePath).Error("Failed to process manifest")
		}
	}

	log.Info("ollama-sync complete.")
}
