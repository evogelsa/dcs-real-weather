package config

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"

	"github.com/evogelsa/DCS-real-weather/weather"
)

//go:embed config.json
var defaultConfig string

// config stores the parsed configuration. Use Get() to retrieve it
var config Configuration

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
			OpenMeteo      bool    `json:"open-meteo"`
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

// ParseConfig reads config.json and returns a Configuration struct of the
// parameters found
func init() {

	file, err := os.Open("config.json")
	if err != nil {
		// if config.json does not exist, create it and exit
		if errors.Is(err, fs.ErrNotExist) {
			log.Println("Config does not exist, creating one...")
			err := os.WriteFile("config.json", []byte(defaultConfig), 0666)
			if err != nil {
				log.Fatalf("Unable to create config.json.")
			}
			log.Fatalf("Default config created. Please configure with your API key and desired settings, then rerun.")
		} else {
			log.Fatalf("Error opening config.json: %v\n", err)
		}
	}

	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatalf("Error decoding config.json: %v\n", err)
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

	// enforce configuration parameters
	checkParams()
}

func checkParams() {
	checkStability()
	checkDefaultPreset()
	checkFogThickness()
	checkVisibility()
	checkDust()
	checkWind()
}

// checkStability enforces stability must be greater than 0
func checkStability() {
	if config.Options.Wind.Stability <= 0 {
		log.Printf(
			"Parsed stability of %0.3f from config file, but stability must be greater than 0.\n",
			config.Options.Wind.Stability,
		)
		log.Println("Stability will default to neutral stability of 0.143.")
		config.Options.Wind.Stability = 0.143
	}
}

// checkDefaultPreset enforces default preset must exist
func checkDefaultPreset() {
	if config.Options.Clouds.DefaultPreset == "" {
		return
	}

	var presetFound bool
	for preset := range weather.DecodePreset {
		if preset == config.Options.Clouds.DefaultPreset {
			presetFound = true
			break
		}
	}

	if !presetFound {
		log.Printf(
			"Default preset %s is not a valid preset. Using clear instead",
			config.Options.Clouds.DefaultPreset,
		)
		config.Options.Clouds.DefaultPreset = ""
	}
}

// checkFogThickness enforces fog thickness within [0, 1000]
func checkFogThickness() {
	if config.Options.Fog.ThicknessMaximum > 1000 {
		log.Println("Fog maximum thickness is set above max of 1000; defaulting to 1000")
		config.Options.Fog.ThicknessMaximum = 1000
	}

	if config.Options.Fog.ThicknessMinimum < 0 {
		log.Println("Fog minimum thickness is set below min of 0; defaulting to 0")
		config.Options.Fog.ThicknessMinimum = 0
	}
}

// checkVisibility enforces vis is within [0, 6000]
func checkVisibility() {
	if config.Options.Fog.VisibilityMaximum > 6000 {
		log.Println("Fog maximum visibility is set above max of 6000; defaulting to 6000")
		config.Options.Fog.VisibilityMaximum = 6000
	}

	if config.Options.Fog.VisibilityMinimum < 0 {
		log.Println("Fog minimum visibility is set below min of 0; defaulting to 0")
		config.Options.Fog.VisibilityMinimum = 0
	}
}

// checkDust enforces dust within [300, 3000]
func checkDust() {
	if config.Options.Dust.VisibilityMinimum < 300 {
		log.Println("Dust visibility minimum is set below min of 300; defaulting to 300")
		config.Options.Dust.VisibilityMinimum = 300
	}

	if config.Options.Dust.VisibilityMaximum > 3000 {
		log.Println("Dust visibility maximum is set above max of 3000; defaulting to 3000")
		config.Options.Dust.VisibilityMaximum = 3000
	}
}

// checkWind enforces wind within [0, 50]
func checkWind() {
	if config.Options.Wind.Minimum < 0 {
		log.Println("Wind minimum is set below min of 0; defaulting to 0")
		config.Options.Wind.Minimum = 0
	}

	if config.Options.Wind.Maximum > 50 {
		log.Println("Wind maximum is set above max of 50; defaulting to 50")
		config.Options.Wind.Maximum = 50
	}
}

func Get() Configuration {
	return config
}

func Set(param string, value interface{}) error {
	switch param {
	case "open-meteo":
		v := value.(bool)
		config.Options.Wind.OpenMeteo = v
	default:
		return fmt.Errorf("Unsupported parameter")
	}
	return nil
}
