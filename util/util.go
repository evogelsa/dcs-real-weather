package util

import (
	"encoding/json"
	"io"
	"log"
	"math"
	"os"
)

// Clamp returns a value that does not exceed the specified range [min, max]
func Clamp(v, min, max float64) float64 {
	v = math.Max(v, min)
	v = math.Min(v, max)
	return v
}

// Configuration is the structure of config.json to be parsed
type Configuration struct {
	APIKey string `json:"api-key"`
	Files  struct {
		InputMission  string `json:"input-mission"`
		OutputMission string `json:"output-mission"`
		Log           string `json:"log"`
	} `json:"files"`
	METAR struct {
		ICAO    string `json:"icao"`
		Remarks string `json:"remarks"`
	} `json:"metar"`
	Options struct {
		UpdateTime    bool   `json:"update-time"`
		UpdateWeather bool   `json:"update-weather"`
		TimeOffset    string `json:"time-offset"`
		Wind          struct {
			Minimum   float64 `json:"minimum"`
			Maximum   float64 `json:"maximum"`
			Stability float64 `json:"stability"`
		} `json:"wind"`
		Clouds struct {
			DisallowedPresets []string `json:"disallowed-presets"`
		}
		Fog struct {
			Enabled           bool `json:"enabled"`
			ThicknessMinimum  int  `json:"thickness-minimum"`
			ThicknessMaximum  int  `json:"thickness-maximum"`
			VisibilityMinimum int  `json:"visibility-minimum"`
			VisibilityMaximum int  `json:"visibility-maximum"`
		} `json:"fog"`
		Dust struct {
			Enabled           bool `json:"enabled"`
			VisibilityMinimum int  `json:"visibility-minimum"`
			VisibilityMaximum int  `json:"visibility-maximum"`
		} `json:"dust"`
	} `json:"options"`
}

var Config Configuration

// ParseConfig reads config.json and returns a Configuration struct of the
// parameters found
func init() {
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

	// stability must be a number greater than 0.
	if config.Options.Wind.Stability <= 0 {
		log.Printf(
			"Parsed stability of %0.3f from config file, but stability must be greater than 0.\n",
			config.Options.Wind.Stability,
		)
		log.Println("Stability will default to neutral stability of 0.143.")
		config.Options.Wind.Stability = 0.143
	}

	if config.Files.Log != "" {
		f, err := os.OpenFile(
			config.Files.Log,
			os.O_WRONLY|os.O_CREATE|os.O_APPEND,
			0644,
		)
		if err != nil {
			log.Printf("Error opening log file: %v\n", err)
		}
		// defer f.Close()

		mw := io.MultiWriter(os.Stdout, f)

		log.SetOutput(mw)
	}

	Config = config
}
