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

// Must panics on error
func Must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Configuration is the structure of config.json to be parsed
type Configuration struct {
	APIKey        string        `json:"api-key"`
	ICAO          string        `json:"icao"`
	HourOffset    time.Duration `json:"hour-offset"`
	InputFile     string        `json:"input-mission-file"`
	OutputFile    string        `json:"output-mission-file"`
	UpdateTime    bool          `json:"update-time"`
	UpdateWeather bool          `json:"update-weather"`
	Logfile       string        `json:"logfile"`
}

// ParseConfig reads config.json and returns a Configuration struct of the
// parameters found
func ParseConfig() {
	var config Configuration
	file, err := os.Open("config.json")
	Must(err)
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	Must(err)

	Config = config

	if Config.Logfile != "" {
		f, err := os.OpenFile(Config.Logfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		Must(err)
		// defer f.Close()

		mw := io.MultiWriter(os.Stdout, f)

		log.SetOutput(mw)
	}
}

var Config Configuration
