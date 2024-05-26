package config

import (
	_ "embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

type Configuration struct {
	GuildID         string `json:"guild-id"`
	BotToken        string `json:"bot-token"`
	RealWeatherPath string `json:"real-weather-path"`
	RealWeatherLog  string `json:"real-weather-log-path"`
	Log             string `json:"log"`
}

//go:embed config.json
var defaultConfig string

const configName = "botconfig.json"

var config Configuration

func init() {
	// open configuration or create default config if not exist
	file, err := os.Open(configName)
	if err != nil {
		// if config.json does not exist, create it and exit
		if errors.Is(err, fs.ErrNotExist) {
			log.Println("Config does not exist, creating one...")
			err := os.WriteFile(configName, []byte(defaultConfig), 0666)
			if err != nil {
				log.Fatalf("Unable to create config: %v", err)
			}
			log.Fatalf("Default config created. Please configure with your API key and desired settings, then rerun.")
		} else {
			log.Fatalf("Error opening config: %v", err)
		}
	}
	defer file.Close()

	// decode configuration
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatalf("Error decoding config: %v", err)
	}

	// remove exe from path
	config.RealWeatherPath = filepath.Dir(config.RealWeatherPath)
}

func Get() Configuration {
	return config
}
