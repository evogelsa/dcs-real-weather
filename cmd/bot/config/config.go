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
	Log             string `json:"log"`
	Instances []struct {
        Name            string `json:"name"`
		RealWeatherPath string `json:"real-weather-path"`
		RealWeatherLog  string `json:"real-weather-log-path"`
	} `json:"instances"`
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

	// verify config values
	var fatalBotConfig bool

	if config.GuildID == "" {
		log.Println("Guild ID is required to be configured.")
		fatalBotConfig = true
	}

	if config.BotToken == "" {
		log.Println("Bot token is required to be configured.")
		fatalBotConfig = true
	}

	if len(config.Instances) < 1 {
		log.Println("At least one instance must be configured.")
		fatalBotConfig = true
	}

	// validate each instance
    names := make(map[string]bool)
	for i := range config.Instances {
        if _, ok := names[config.Instances[i].Name]; ok {
            log.Printf("Name is required to be unique for each instance (check instance #%d).", i+1)
            fatalBotConfig = true
        }

        if config.Instances[i].Name == "" {
            log.Printf("Name is required to be configured for each instance (check instance #%d).", i+1)
            fatalBotConfig = true
        }

        names[config.Instances[i].Name] = true

		config.Instances[i].RealWeatherPath = filepath.Dir(config.Instances[i].RealWeatherPath)

		if config.Instances[i].RealWeatherPath == "" {
			log.Printf("Real Weather path is required to be configured for each instance (check instance #%d).", i+1)
			fatalBotConfig = true
		}

		if config.Instances[i].RealWeatherLog == "" {
			log.Printf("Real Weather log path is required to be configured for each instance (check instance #%d).", i+1)
			fatalBotConfig = true
		}
	}

	if fatalBotConfig {
		log.Fatalf("One or more errors exist in your config. Please correct them then restart the bot!")
	}

}

func Get() Configuration {
	return config
}
