package util

import (
	"encoding/json"
	"io"
	"log"
	"math"
	"os"
	"time"
)

// Clamp returns a value that does not exceed the specified range [min, max]
func Clamp(v, min, max float64) float64 {
	v = math.Max(v, min)
	v = math.Min(v, max)
	return v
}

// Configuration is the structure of config.json to be parsed
type Configuration struct {
	APIKey        string        `json:"api-key"`
	ICAO          string        `json:"icao"`
	HourOffset    time.Duration `json:"hour-offset"`
	Stability     float64       `json:"stability"`
	InputFile     string        `json:"input-mission-file"`
	OutputFile    string        `json:"output-mission-file"`
	UpdateTime    bool          `json:"update-time"`
	UpdateWeather bool          `json:"update-weather"`
	Logfile       string        `json:"logfile"`
	Remarks       string        `json:"metar-remarks"`
}

// ParseConfig reads config.json and returns a Configuration struct of the
// parameters found
func ParseConfig() {
	var config Configuration
	file, err := os.Open("config.json")
	if err != nil {
		log.Fatalf("Error opening config.json: %v\n", err)
	}

	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatalf("Error decoding config.json: %v\n", err)
	}

	Config = config

	if Config.Logfile != "" {
		f, err := os.OpenFile(
			Config.Logfile,
			os.O_WRONLY|os.O_CREATE|os.O_APPEND,
			0644,
		)
		if err != nil {
			log.Printf("Error opening logfile: %v\n", err)
		}
		// defer f.Close()

		mw := io.MultiWriter(os.Stdout, f)

		log.SetOutput(mw)
	}
}

var Config Configuration
