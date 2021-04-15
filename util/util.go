package util

import (
	"encoding/json"
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
		panic(err)
	}
}

// Configuration is the structure of config.json to be parsed
type Configuration struct {
	APIKey     string        `json:"api-key"`
	ICAO       string        `json:"icao"`
	HourOffset time.Duration `json:"hour-offset"`
	InputFile  string        `json:"input-mission-file"`
	OutputFile string        `json:"output-mission-file"`
}

// ParseConfig reads config.json and returns a Configuration struct of the
// parameters found
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
