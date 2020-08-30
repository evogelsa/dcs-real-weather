package util

import (
	"encoding/json"
	"os"
)

// Must performs a lazy error "check"
func Must(err error) {
	if err != nil {
		panic(err)
	}
}

// Configuration is the structure of config.json to be parsed
type Configuration struct {
	APIKey     string `json:"api-key"`
	ICAO       string `json:"icao"`
	HourOffset int    `json:"hour-offset"`
	InputFile  string `json:"input-mission-file"`
	OutputFile string `json:"output-mission-file"`
}

func ParseConfig() Configuration {
	var config Configuration
	file, err := os.Open("config.json")
	Must(err)
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	Must(err)

	return config
}
