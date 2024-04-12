package util

import (
	"encoding/json"
	"io"
	"log"
	"math"
	"os"

	"golang.org/x/exp/constraints"

	"github.com/evogelsa/DCS-real-weather/weather"
)

// Clamp returns a value that does not exceed the specified range [min, max]
func Clamp[T1, T2, T3 constraints.Float | constraints.Integer](v T1, min T2, max T3) T1 {
	v = T1(math.Max(float64(v), float64(min)))
	v = T1(math.Min(float64(v), float64(max)))
	return v
}

// Between returns if a value is between the specified range [min, max]
func Between[T1, T2, T3 constraints.Float | constraints.Integer](v T1, min T2, max T3) bool {
	return float64(min) <= float64(v) && float64(v) <= float64(max)
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
		ICAO            string `json:"icao"`
		RunwayElevation int    `json:"runway-elevation"`
		Remarks         string `json:"remarks"`
		AddToBrief      bool   `json:"add-to-brief"`
	} `json:"metar"`
	Options struct {
		UpdateTime    bool   `json:"update-time"`
		UpdateWeather bool   `json:"update-weather"`
		TimeOffset    string `json:"time-offset"`
		Wind          struct {
			Minimum        float64 `json:"minimum"`
			Maximum        float64 `json:"maximum"`
			Stability      float64 `json:"stability"`
			FixedReference bool    `json:"fixed-reference"`
		} `json:"wind"`
		Clouds struct {
			DisallowedPresets  []string `json:"disallowed-presets"`
			FallbackToNoPreset bool     `json:"fallback-to-no-preset"`
			DefaultPreset      string   `json:"default-preset"`
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

	// default preset must exist in list of presets
	var presetFound bool
	for preset := range weather.DecodePreset {
		if preset == Config.Options.Clouds.DefaultPreset {
			presetFound = true
			break
		}
	}

	if !presetFound {
		log.Printf(
			"Default preset %s is not a valid preset. Using clear instead",
			Config.Options.Clouds.DefaultPreset,
		)
		Config.Options.Clouds.DefaultPreset = ""
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
