package config

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"regexp"
	"slices"
	"time"

	"github.com/pelletier/go-toml/v2"

	"github.com/evogelsa/DCS-real-weather/util"
	"github.com/evogelsa/DCS-real-weather/weather"
)

// Configuration is the structure of config.json to be parsed
type Configuration struct {
	RealWeather struct {
		Mission struct {
			Input  string `toml:"input"`
			Output string `toml:"output"`
			Brief  struct {
				AddMETAR  bool   `toml:"add-metar"`
				InsertKey string `toml:"insert-key"`
				Remarks   string `toml:"remarks"`
			} `toml:"brief"`
		} `toml:"mission"`
		Log struct {
			Enable bool   `toml:"enable"`
			File   string `toml:"file"`
		} `toml:"log"`
	} `toml:"realweather"`
	API struct {
		ProviderPriority []string `toml:"provider-priority"`
		AviationWeather  struct {
			Enable bool `toml:"enable"`
		} `toml:"aviationweather"`
		CheckWX struct {
			Enable bool   `toml:"enable"`
			Key    string `toml:"key"`
		} `toml:"checkwx"`
		Custom struct {
			Enable   bool   `toml:"enable"`
			File     string `toml:"file"`
			Override bool   `toml:"override"`
		} `toml:"custom"`
		OpenMeteo struct {
			Enable bool `toml:"enable"`
		} `toml:"openmeteo"`
	} `toml:"api"`
	Options struct {
		Time struct {
			Enable     bool   `toml:"enable"`
			SystemTime bool   `toml:"system-time"`
			Offset     string `toml:"offset"`
		} `toml:"time"`
		Date struct {
			Enable     bool   `toml:"enable"`
			SystemDate bool   `toml:"system-date"`
			Offset     string `toml:"offset"`
		} `toml:"date"`
		Weather struct {
			Enable          bool     `toml:"enable"`
			ICAO            string   `toml:"icao"`
			ICAOList        []string `toml:"icao-list"`
			RunwayElevation float64  `toml:"runway-elevation"`
			Wind            struct {
				Enable         bool    `toml:"enable"`
				Minimum        float64 `toml:"minimum"`
				Maximum        float64 `toml:"maximum"`
				GustMinimum    float64 `toml:"gust-minimum"`
				GustMaximum    float64 `toml:"gust-maximum"`
				Stability      float64 `toml:"stability"`
				FixedReference bool    `toml:"fixed-reference"`
			} `toml:"wind"`
			Clouds struct {
				Enable           bool `toml:"enable"`
				FallbackToLegacy bool `toml:"fallback-to-legacy"`
				Base             struct {
					Minimum float64 `toml:"minimum"`
					Maximum float64 `toml:"maximum"`
				} `toml:"base"`
				Presets struct {
					Default    string   `toml:"default"`
					Disallowed []string `toml:"disallowed"`
				} `toml:"presets"`
			} `toml:"clouds"`
			Fog struct {
				Enable            bool    `toml:"enable"`
				Mode              string  `toml:"mode"`
				ThicknessMinimum  float64 `toml:"thickness-minimum"`
				ThicknessMaximum  float64 `toml:"thickness-maximum"`
				VisibilityMinimum float64 `toml:"visibility-minimum"`
				VisibilityMaximum float64 `toml:"visibility-maximum"`
			} `toml:"fog"`
			Dust struct {
				Enable            bool    `toml:"enable"`
				VisibilityMinimum float64 `toml:"visibility-minimum"`
				VisibilityMaximum float64 `toml:"visibility-maximum"`
			}
			Temperature struct {
				Enable bool `toml:"enable"`
			} `toml:"temperature"`
			Pressure struct {
				Enable bool `toml:"enable"`
			} `toml:"pressure"`
		}
	}
}

// Overrideable defines values of the config which can be overridden through
// command line interface
type Overrideable struct {
	APICustomEnable    bool
	APICustomFile      string
	MissionInput       string
	MissionOutput      string
	OptionsWeatherICAO string
}

// config stores the parsed configuration. Use Get() to retrieve it
var config Configuration

//go:embed config.toml
var defaultConfig string

// Init reads config.toml and umarshals into config
func Init(configName string, overrides Overrideable) {
	log.Printf("Reading %s", configName)

	file, err := os.Open(configName)
	if err != nil {
		// if config.toml does not exist, create it and exit
		if errors.Is(err, fs.ErrNotExist) {
			log.Println("Config does not exist, creating one...")
			err := os.WriteFile(configName, []byte(defaultConfig), 0666)
			if err != nil {
				log.Fatalf("Unable to create %s: %v", configName, err)
			}
			log.Fatalf("Default config created. Please configure with your desired settings, then rerun.")
		} else {
			log.Fatalf("Error opening %s: %v\n", configName, err)
		}
	}

	defer file.Close()
	decoder := toml.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatalf("Error decoding %s: %v\n", configName, err)
	}

	if config.RealWeather.Log.Enable {
		f, err := os.OpenFile(
			config.RealWeather.Log.File,
			os.O_WRONLY|os.O_CREATE|os.O_APPEND,
			0644,
		)
		if err != nil {
			log.Printf("Error opening log file %s: %v\n", config.RealWeather.Log.File, err)
		}
		// defer f.Close()

		mw := io.MultiWriter(os.Stdout, f)

		log.SetOutput(mw)
	}

	// apply overrides
	if overrides.APICustomEnable {
		config.API.Custom.Enable = overrides.APICustomEnable
	}

	if overrides.APICustomFile != "" {
		config.API.Custom.File = overrides.APICustomFile
	}

	if overrides.MissionInput != "" {
		config.RealWeather.Mission.Input = overrides.MissionInput
	}

	if overrides.MissionOutput != "" {
		config.RealWeather.Mission.Output = overrides.MissionOutput
	}

	if overrides.OptionsWeatherICAO != "" {
		config.Options.Weather.ICAO = overrides.OptionsWeatherICAO
	}

	// enforce configuration parameters
	log.Println("Validating configuration")
	checkParams()
	log.Println("Configuration validated")
}

func Get() Configuration {
	return config
}

func Set(param string, value interface{}) error {
	switch param {
	case "open-meteo":
		v := value.(bool)
		config.API.OpenMeteo.Enable = v
	case "input":
		v := value.(string)
		config.RealWeather.Mission.Input = v
	case "output":
		v := value.(string)
		config.RealWeather.Mission.Output = v
	default:
		return fmt.Errorf("Unsupported parameter")
	}
	return nil
}

// checkParams calls the config checking functions
func checkParams() {
	checkRealWeather()
	checkAPI()
	checkOptionsTime()
	checkOptionsWeather()
	checkOptionsWind()
	checkOptionsClouds()
	checkOptionsFog()
	checkOptionsDust()
}

// checkRealWeather validates the realweather section of the config
func checkRealWeather() {
	var fatal bool
	if config.RealWeather.Mission.Input == "" {
		log.Println("No input mission configured, one must be supplied")
		fatal = true
	}
	if config.RealWeather.Mission.Output == "" {
		log.Println("No output mission configured, one must be supplied")
		fatal = true
	}
	if _, err := regexp.Compile(config.RealWeather.Mission.Brief.InsertKey); err != nil {
		log.Printf("Brief insert key must be valid when used in a PCRE regular expression: %v", err)
		fatal = true
	}
	if fatal {
		log.Fatalln("Irrecoverable errors found in config, please correct these and try again.")
	}
}

// checkAPI validates all the API settings in the config
func checkAPI() {
	// if checkwx is enabled, validate that a key is present
	if config.API.CheckWX.Enable && config.API.CheckWX.Key == "" {
		log.Println("API key for CheckWX is missing. CheckWX will be disabled")
		config.API.CheckWX.Enable = false
	}

	// validate at least one provider is enabled
	if !config.API.AviationWeather.Enable &&
		!config.API.CheckWX.Enable &&
		!config.API.Custom.Enable {
		log.Println("All providers are disabled, aviationweather will be enabled by default")
		config.API.AviationWeather.Enable = true
	}

	// verify providers are valid
	knownProviders := []weather.API{
		weather.APIAviationWeather,
		weather.APICheckWX,
		weather.APICustom,
	}
	for _, provider := range config.API.ProviderPriority {
		if !slices.Contains(knownProviders, weather.API(provider)) {
			log.Printf("Provider %s is not a recognized API; it will be ignored", provider)
		}
	}

	// ensure each provider is in priority list
	for _, provider := range knownProviders {
		if !slices.Contains(config.API.ProviderPriority, string(provider)) {
			log.Printf("Provider %s missing from priority list, adding to end", provider)
			config.API.ProviderPriority = append(config.API.ProviderPriority, string(provider))
		}
	}
}

// checkOptionsTime validates the time options in the config
func checkOptionsTime() {
	_, err := time.ParseDuration(config.Options.Time.Offset)
	if err != nil {
		log.Printf(
			"Could not parse time offset of %s: %v",
			config.Options.Time.Offset,
			err,
		)
		log.Println("Time offset will default to zero")
		config.Options.Time.Offset = "0"
	}
}

// checkOptionsDate validates the date options in the config
func checkOptionsDate() {
	_, err := util.ParseDateDuration(config.Options.Date.Offset)
	if err != nil {
		log.Printf(
			"Could not parse date offset of %s: %v",
			config.Options.Date.Offset,
			err,
		)
		log.Println("Date offset will default to zero")
		config.Options.Date.Offset = "0"
	}
}

// checkOptionsWeather valides the weather options in the config
func checkOptionsWeather() {
	// validate ICAOs given are valid format (doesn't check if actually exists)
	re := regexp.MustCompile("^[A-Z]{4}$")
	for i, icao := range config.Options.Weather.ICAOList {
		if !re.MatchString(icao) {
			log.Printf("ICAO %s in ICAO list is not a valid ICAO and will not be used", icao)
			config.Options.Weather.ICAOList = slices.Delete(
				config.Options.Weather.ICAOList,
				i,
				i+1,
			)
		}
	}

	if config.Options.Weather.ICAO != "" {
		if !re.MatchString(config.Options.Weather.ICAO) {
			log.Printf("ICAO %s is not a valid ICAO and will not be used", config.Options.Weather.ICAO)
			config.Options.Weather.ICAO = ""
		}
	}

	// validate an option for ICAO exists
	if config.Options.Weather.ICAO == "" && len(config.Options.Weather.ICAOList) == 0 {
		log.Println("ICAO or ICAO list must be supplied, defaulting to ICAO to DGAA")
		config.Options.Weather.ICAO = "DGAA"
	} else if config.Options.Weather.ICAO != "" && len(config.Options.Weather.ICAOList) > 0 {
		log.Println("ICAO and ICAO list cannot be used at the same time, only ICAO will be used")
	}
}

// checkOptionsWind validates wind options in the config
func checkOptionsWind() {
	if config.Options.Weather.Wind.Minimum < 0 {
		log.Println("Wind minimum is set below min of 0; defaulting to 0")
		config.Options.Weather.Wind.Minimum = 0
	}

	if config.Options.Weather.Wind.Maximum > 50 {
		log.Println("Wind maximum is set above max of 50; defaulting to 50")
		config.Options.Weather.Wind.Maximum = 50
	}

	if config.Options.Weather.Wind.Minimum > config.Options.Weather.Wind.Maximum {
		log.Println("Wind minimum is set above wind maximum; defaulting to 0 and 50")
		config.Options.Weather.Wind.Minimum = 50
		config.Options.Weather.Wind.Maximum = 0
	}

	if config.Options.Weather.Wind.GustMinimum < 0 {
		log.Println("Gust minimum is set below min of 0; defaulting to 0")
		config.Options.Weather.Wind.GustMinimum = 0
	}

	if config.Options.Weather.Wind.GustMaximum > 50 {
		log.Println("Gust maximum is set above max of 50; defaulting to 50")
		config.Options.Weather.Wind.GustMaximum = 50
	}

	if config.Options.Weather.Wind.GustMinimum > config.Options.Weather.Wind.GustMaximum {
		log.Println("Gust minimum is set above gust maximum; defaulting to 0 and 50")
		config.Options.Weather.Wind.Maximum = 50
		config.Options.Weather.Wind.Minimum = 0
	}

	if config.Options.Weather.Wind.Stability <= 0 {
		log.Printf(
			"Parsed stability of %0.3f from config file, but stability must be greater than 0.\n",
			config.Options.Weather.Wind.Stability,
		)
		log.Println("Stability will default to neutral stability of 0.143.")
		config.Options.Weather.Wind.Stability = 0.143
	}
}

// checkOptionsClouds validates cloud options in the config
func checkOptionsClouds() {
	if config.Options.Weather.Clouds.Base.Minimum < 0 {
		log.Printf("Minimum cloud base must be >=0; it will default to 0")
		config.Options.Weather.Clouds.Base.Minimum = 0
	}

	if config.Options.Weather.Clouds.Base.Maximum > 15000 {
		log.Printf("Maximum cloud base must be <=15000; it will default to 15000")
		config.Options.Weather.Clouds.Base.Maximum = 15000
	}

	var presetFound bool
	var presetMinBase int
	var presetMaxBase int
	if config.Options.Weather.Clouds.Presets.Default != "" {
		for preset := range weather.DecodePreset {
			if preset == `"`+config.Options.Weather.Clouds.Presets.Default+`"` {
				presetFound = true
				break
			}
		}

		// search for default preset min and max base
		for _, presetList := range weather.CloudPresets {
			for _, preset := range presetList {
				if preset.Name == config.Options.Weather.Clouds.Presets.Default {
					presetMinBase = preset.MinBase
					presetMaxBase = preset.MaxBase
				}
			}
		}
	} else {
		presetFound = true
		presetMinBase = int(config.Options.Weather.Clouds.Base.Minimum)
		presetMaxBase = int(config.Options.Weather.Clouds.Base.Minimum)
	}

	if !presetFound {
		log.Printf(
			"Default preset %s is not a valid preset. Using clear instead",
			config.Options.Weather.Clouds.Presets.Default,
		)
		config.Options.Weather.Clouds.Presets.Default = ""
	}

	// check that default preset min/max base falls within config min/max
	if config.Options.Weather.Clouds.Base.Minimum > float64(presetMaxBase) {
		log.Print(
			"The configured default preset has a max base lower than the configured min base." +
				" This value may be ignored if the default preset is used.",
		)
	}
	if config.Options.Weather.Clouds.Base.Maximum < float64(presetMinBase) {
		log.Print(
			"The configured default preset has a min base lower than the configured max base." +
				" This value may be ignored if the default preset is used.",
		)
	}

	for _, preset := range config.Options.Weather.Clouds.Presets.Disallowed {
		presetFound = false
		for valid := range weather.DecodePreset {
			if valid == `"`+preset+`"` {
				presetFound = true
				break
			}
		}
		if !presetFound {
			log.Printf(
				"Preset %s in disallowed list is not a valid preset; it'll be ignored",
				preset,
			)
		}
	}
}

// checkOptionsFog enforces fog configuration options
func checkOptionsFog() {
	if config.Options.Weather.Fog.Mode != string(weather.FogAuto) &&
		config.Options.Weather.Fog.Mode != string(weather.FogManual) &&
		config.Options.Weather.Fog.Mode != string(weather.FogLegacy) {
		log.Println("Fog mode unrecognized (expecting \"auto\", \"manual\", or \"legacy\"); defaulting to auto")
		config.Options.Weather.Fog.Mode = string(weather.FogAuto)
	}

	if config.Options.Weather.Fog.ThicknessMaximum > 1000 {
		log.Println("Fog maximum thickness is set above max of 1000; defaulting to 1000")
		config.Options.Weather.Fog.ThicknessMaximum = 1000
	}

	if config.Options.Weather.Fog.ThicknessMinimum < 0 {
		log.Println("Fog minimum thickness is set below min of 0; defaulting to 0")
		config.Options.Weather.Fog.ThicknessMinimum = 0
	}

	if config.Options.Weather.Fog.ThicknessMinimum > config.Options.Weather.Fog.ThicknessMaximum {
		log.Println("Fog minimum thickness is set above fog maximum thickness; defaulting to 0 and 1000")
		config.Options.Weather.Fog.ThicknessMaximum = 1000
		config.Options.Weather.Fog.ThicknessMinimum = 0
	}

	if config.Options.Weather.Fog.VisibilityMaximum > 6000 {
		log.Println("Fog maximum visibility is set above max of 6000; defaulting to 6000")
		config.Options.Weather.Fog.VisibilityMaximum = 6000
	}

	if config.Options.Weather.Fog.VisibilityMinimum < 0 {
		log.Println("Fog minimum visibility is set below min of 0; defaulting to 0")
		config.Options.Weather.Fog.VisibilityMinimum = 0
	}

	if config.Options.Weather.Fog.VisibilityMinimum > config.Options.Weather.Fog.VisibilityMaximum {
		log.Println("Fog minimum visibility is set above fog maximum visibility; defaulting to 0 and 6000")
		config.Options.Weather.Fog.VisibilityMaximum = 6000
		config.Options.Weather.Fog.VisibilityMinimum = 0
	}
}

// checkOptionsDust enforces dust configuration options
func checkOptionsDust() {
	if config.Options.Weather.Dust.VisibilityMinimum < 300 {
		log.Println("Dust visibility minimum is set below min of 300; defaulting to 300")
		config.Options.Weather.Dust.VisibilityMinimum = 300
	}

	if config.Options.Weather.Dust.VisibilityMaximum > 3000 {
		log.Println("Dust visibility maximum is set above max of 3000; defaulting to 3000")
		config.Options.Weather.Dust.VisibilityMaximum = 3000
	}

	if config.Options.Weather.Dust.VisibilityMinimum > config.Options.Weather.Fog.VisibilityMaximum {
		log.Println("Dust minimum visibility is set above dust maximum visibility; defaulting to 300 and 3000")
		config.Options.Weather.Dust.VisibilityMaximum = 3000
		config.Options.Weather.Dust.VisibilityMinimum = 300
	}
}
