package config

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"regexp"
	"slices"
	"time"

	"github.com/pelletier/go-toml/v2"

	"github.com/evogelsa/DCS-real-weather/logger"
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
			Enable     bool   `toml:"enable"`
			File       string `toml:"file"`
			MaxSize    int    `toml:"max-size"`
			MaxBackups int    `toml:"max-backups"`
			MaxAge     int    `toml:"max-age"`
			Compress   bool   `toml:"compress"`
			Level      string `toml:"level"`
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
				Enable bool `toml:"enable"`
				Base   struct {
					Minimum float64 `toml:"minimum"`
					Maximum float64 `toml:"maximum"`
				} `toml:"base"`
				Presets struct {
					Default    string   `toml:"default"`
					Disallowed []string `toml:"disallowed"`
				} `toml:"presets"`
				Custom struct {
					Enable             bool    `toml:"enable"`
					AllowPrecipitation bool    `toml:"allow-precipitation"`
					DensityMinimum     float64 `toml:"density-minimum"`
					DensityMaximum     float64 `toml:"density-maximum"`
				} `toml:"custom"`
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
	err := toml.Unmarshal([]byte(defaultConfig), &config)
	if err != nil {
		log.Fatalf("unable to read default config")
	}

	file, err := os.Open(configName)
	if err != nil {
		// if config.toml does not exist, create it and exit
		if errors.Is(err, fs.ErrNotExist) {
			log.Println("config does not exist, creating one...")
			err := os.WriteFile(configName, []byte(defaultConfig), 0666)
			if err != nil {
				log.Fatalf("unable to create %s: %v", configName, err)
			}
			log.Println("default config created")
			log.Println("please configure with your desired settings then rerun real weather")
			log.Println("see https://github.com/evogelsa/dcs-real-weather for more information")
			os.Exit(0)
		} else {
			log.Fatalf("error opening %s: %v", configName, err)
		}
	}

	defer file.Close()
	decoder := toml.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatalf("error decoding %s: %v", configName, err)
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
		return fmt.Errorf("unsupported parameter")
	}
	return nil
}

// Validate calls the config checking functions
func Validate() {
	logger.Infoln("validating configuration")
	checkRealWeather()
	checkAPI()
	checkOptionsTime()
	checkOptionsWeather()
	checkOptionsWind()
	checkOptionsClouds()
	checkOptionsFog()
	checkOptionsDust()
	logger.Infoln("configuration validated")
}

// checkRealWeather validates the realweather section of the config
func checkRealWeather() {
	var fatal bool

	if config.RealWeather.Mission.Input == "" {
		logger.Errorln("no input mission configured")
		fatal = true
	}

	if config.RealWeather.Mission.Output == "" {
		logger.Errorln("no output mission configured")
		fatal = true
	}

	if _, err := regexp.Compile(config.RealWeather.Mission.Brief.InsertKey); err != nil {
		logger.Errorf("brief insert key must be a valid go regexp: %v", err)
		fatal = true
	}

	if config.RealWeather.Log.MaxSize < 0 {
		logger.Errorf("log max size is <0")
		config.RealWeather.Log.MaxSize = 0
		logger.Warnln("log max size defaulted to 0 (disabled)")
	}

	if config.RealWeather.Log.MaxBackups < 0 {
		logger.Errorf("log max backups is <0")
		config.RealWeather.Log.MaxBackups = 0
		logger.Warnln("log max backups defaulted to 0 (disabled)")
	}

	if config.RealWeather.Log.MaxAge < 0 {
		logger.Errorf("log max age is <0")
		config.RealWeather.Log.MaxAge = 0
		logger.Warnln("log max age defaulted to 0 (disabled)")
	}

	if config.RealWeather.Log.Level != "debug" &&
		config.RealWeather.Log.Level != "info" &&
		config.RealWeather.Log.Level != "warn" &&
		config.RealWeather.Log.Level != "error" {
		// log level validation happens before logger.Init is called, just show
		// the error here (so we can use the logger)
		logger.Errorf("log level \"%s\" is unrecognized", config.RealWeather.Log.Level)
		logger.Warnln("log level defaulted to \"info\"")
	}

	if fatal {
		logger.Fatalln("irrecoverable errors found in config")
	}
}

// checkAPI validates all the API settings in the config
func checkAPI() {
	// if checkwx is enabled, validate that a key is present
	if config.API.CheckWX.Enable && config.API.CheckWX.Key == "" {
		logger.Errorln("checkwx enabled but missing api key")
		config.API.CheckWX.Enable = false
		logger.Warnln("checkwx disabled")
	}

	// validate at least one provider is enabled
	if !config.API.AviationWeather.Enable &&
		!config.API.CheckWX.Enable &&
		!config.API.Custom.Enable {
		logger.Errorln("all providers are disabled")
		config.API.AviationWeather.Enable = true
		logger.Warnln("aviationweather enabled by default")
	}

	// verify providers are valid
	knownProviders := []weather.API{
		weather.APIAviationWeather,
		weather.APICheckWX,
		weather.APICustom,
	}
	for _, provider := range config.API.ProviderPriority {
		if !slices.Contains(knownProviders, weather.API(provider)) {
			logger.Warnf("provider \"%s\" not recognized: ignored", provider)
		}
	}

	// ensure each provider is in priority list
	for _, provider := range knownProviders {
		if !slices.Contains(config.API.ProviderPriority, string(provider)) {
			logger.Errorf("provider \"%s\" missing from priority list", provider)
			config.API.ProviderPriority = append(config.API.ProviderPriority, string(provider))
			logger.Warnf("provider \"%s\" added to end of priority list", provider)
		}
	}
}

// checkOptionsTime validates the time options in the config
func checkOptionsTime() {
	_, err := time.ParseDuration(config.Options.Time.Offset)
	if err != nil {
		logger.Errorf(
			"could not parse time offset \"%s\": %v",
			config.Options.Time.Offset,
			err,
		)
		config.Options.Time.Offset = "0"
		logger.Warnln("time offset defaulted to \"0\"")
	}
}

// checkOptionsDate validates the date options in the config
func checkOptionsDate() {
	_, err := util.ParseDateDuration(config.Options.Date.Offset)
	if err != nil {
		logger.Errorf(
			"could not parse date offset \"%s\": %v",
			config.Options.Date.Offset,
			err,
		)
		config.Options.Date.Offset = "0"
		logger.Warnln("date offset defaulted to \"0\"")
	}
}

// checkOptionsWeather valides the weather options in the config
func checkOptionsWeather() {
	// validate ICAOs given are valid format (doesn't check if actually exists)
	re := regexp.MustCompile("^[A-Z]{4}$")
	for i, icao := range config.Options.Weather.ICAOList {
		if !re.MatchString(icao) {
			logger.Errorf("\"%s\" is not a valid airport code", icao)
			config.Options.Weather.ICAOList = slices.Delete(
				config.Options.Weather.ICAOList,
				i,
				i+1,
			)
			logger.Warnln("ignoring \"%s\" in icao-list")
		}
	}

	if config.Options.Weather.ICAO != "" {
		if !re.MatchString(config.Options.Weather.ICAO) {
			logger.Errorf("\"%s\" is not a valid airport code", config.Options.Weather.ICAO)
			config.Options.Weather.ICAO = ""
		}
	}

	// validate an option for ICAO exists
	if config.Options.Weather.ICAO == "" && len(config.Options.Weather.ICAOList) == 0 {
		logger.Errorln("icao or icao-list must be supplied")
		config.Options.Weather.ICAO = "UGKO"
		logger.Warnln("using UGKO as icao by default")
	} else if config.Options.Weather.ICAO != "" && len(config.Options.Weather.ICAOList) > 0 {
		logger.Warnln("icao and icao-list cannot be used simultaneously (only icao will be used)")
	}
}

// checkOptionsWind validates wind options in the config
func checkOptionsWind() {
	if config.Options.Weather.Wind.Minimum < 0 {
		logger.Errorf("wind minimum %f is below 0", config.Options.Weather.Wind.Minimum)
		config.Options.Weather.Wind.Minimum = 0
		logger.Warnln("wind minimum defaulted to 0")
	}

	if config.Options.Weather.Wind.Maximum > 50 {
		logger.Errorf("wind maximum %f is above 50", config.Options.Weather.Wind.Maximum)
		config.Options.Weather.Wind.Maximum = 50
		logger.Warnln("wind maximum defaulted to 50")
	}

	if config.Options.Weather.Wind.Minimum > config.Options.Weather.Wind.Maximum {
		logger.Errorf("wind minimum %f is greater than wind maximum %f", config.Options.Weather.Wind.Minimum, config.Options.Weather.Wind.Maximum)
		config.Options.Weather.Wind.Minimum = 50
		config.Options.Weather.Wind.Maximum = 0
		logger.Warnln("wind minimum defaulted to 0")
		logger.Warnln("wind maximum defaulted to 50")
	}

	if config.Options.Weather.Wind.GustMinimum < 0 {
		logger.Errorf("gust minimum %f is below 0", config.Options.Weather.Wind.GustMinimum)
		config.Options.Weather.Wind.GustMinimum = 0
		logger.Warnln("gust minimum defaulted to 0")
	}

	if config.Options.Weather.Wind.GustMaximum > 50 {
		logger.Errorf("gust maximum %f is above 50")
		config.Options.Weather.Wind.GustMaximum = 50
		logger.Warnln("gust maximum defaulted to 50")
	}

	if config.Options.Weather.Wind.GustMinimum > config.Options.Weather.Wind.GustMaximum {
		logger.Errorf("gust minimum %f is greater than gust maximum %f", config.Options.Weather.Wind.GustMinimum, config.Options.Weather.Wind.GustMaximum)
		config.Options.Weather.Wind.GustMinimum = 0
		config.Options.Weather.Wind.GustMaximum = 50
		logger.Warnln("gust minimum defaulted to 0")
		logger.Warnln("gust maximum defaulted to 50")
	}

	if config.Options.Weather.Wind.Stability <= 0 {
		logger.Errorf("stability %f must be >0", config.Options.Weather.Wind.Stability)
		config.Options.Weather.Wind.Stability = 0.143
		logger.Warnln("stability defaulted to 0.143")
	}
}

// checkOptionsClouds validates cloud options in the config
func checkOptionsClouds() {
	if config.Options.Weather.Clouds.Base.Minimum < 0 {
		logger.Errorf("minimum cloud base %f must be >=0", config.Options.Weather.Clouds.Base.Minimum)
		config.Options.Weather.Clouds.Base.Minimum = 0
		logger.Warnln("minimum cloud base defaulted to 0")
	}

	if config.Options.Weather.Clouds.Base.Maximum > 15000 {
		logger.Errorf("maximum cloud base %f must be <=15000", config.Options.Weather.Clouds.Base.Maximum)
		config.Options.Weather.Clouds.Base.Maximum = 15000
		logger.Warnln("maximum cloud base defaulted to 15000")
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
		logger.Errorf(
			"default preset \"%s\" is not a valid preset",
			config.Options.Weather.Clouds.Presets.Default,
		)
		config.Options.Weather.Clouds.Presets.Default = ""
		logger.Warnln("default preset defaulted to clear")
	}

	// check that default preset min/max base falls within config min/max
	if config.Options.Weather.Clouds.Base.Minimum > float64(presetMaxBase) {
		logger.Warnln("configured min base is higher than default preset's max base and may be ignored")
	}
	if config.Options.Weather.Clouds.Base.Maximum < float64(presetMinBase) {
		logger.Warnln("configured max base is lower than default preset's min base and may be ignored")
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
			logger.Errorf("disallowed preset \"%s\" is not a valid preset", preset)
			logger.Warnln("ignoring disallowed preset \"%s\"", preset)
		}
	}

	if !util.Between(
		config.Options.Weather.Clouds.Custom.DensityMinimum,
		0,
		config.Options.Weather.Clouds.Custom.DensityMaximum,
	) {
		logger.Errorf("cloud density minimum %f is not between 0 and cloud density maximum", config.Options.Weather.Clouds.Custom.DensityMinimum)
		config.Options.Weather.Clouds.Custom.DensityMinimum = 0
		logger.Warnln("cloud density minimum defaulted to 0")
	}

	if !util.Between(
		config.Options.Weather.Clouds.Custom.DensityMaximum,
		config.Options.Weather.Clouds.Custom.DensityMinimum,
		10,
	) {
		logger.Errorf("cloud density maximum %f is not between 10 and cloud density minimum", config.Options.Weather.Clouds.Custom.DensityMaximum)
		config.Options.Weather.Clouds.Custom.DensityMaximum = 10
		logger.Warnln("cloud density maximum defaulted to 10")
	}
}

// checkOptionsFog enforces fog configuration options
func checkOptionsFog() {
	if config.Options.Weather.Fog.Mode != string(weather.FogAuto) &&
		config.Options.Weather.Fog.Mode != string(weather.FogManual) &&
		config.Options.Weather.Fog.Mode != string(weather.FogLegacy) {
		logger.Errorf("fog mode \"%s\" unrecognized (expecting \"auto\", \"manual\", or \"legacy\")", config.Options.Weather.Fog.Mode)
		config.Options.Weather.Fog.Mode = string(weather.FogAuto)
		logger.Warnln("fog mode defaulted to auto")
	}

	if config.Options.Weather.Fog.ThicknessMinimum < 0 {
		logger.Errorf("fog minimum thickness %f is <0", config.Options.Weather.Fog.ThicknessMinimum)
		config.Options.Weather.Fog.ThicknessMinimum = 0
		logger.Warnln("fog minimum thickness defaulted to 0")
	}

	if config.Options.Weather.Fog.ThicknessMaximum > 1000 {
		logger.Errorf("fog maximum thickness %f is >1000", config.Options.Weather.Fog.ThicknessMaximum)
		config.Options.Weather.Fog.ThicknessMaximum = 1000
		logger.Warnln("fog maxmimum thickness defaulted to 1000")
	}

	if config.Options.Weather.Fog.ThicknessMinimum > config.Options.Weather.Fog.ThicknessMaximum {
		logger.Errorf("fog minimum thickness is greater than fog maximum thickness")
		config.Options.Weather.Fog.ThicknessMaximum = 1000
		config.Options.Weather.Fog.ThicknessMinimum = 0
		logger.Warnln("fog minimum thickness defaulted to 0")
		logger.Warnln("fog maxmimum thickness defaulted to 1000")
	}

	if config.Options.Weather.Fog.VisibilityMinimum < 0 {
		logger.Errorf("fog minimum visibility %f is <0", config.Options.Weather.Fog.VisibilityMinimum)
		config.Options.Weather.Fog.VisibilityMinimum = 0
		logger.Warnln("fog minimum visibility defaulted to 0")
	}

	if config.Options.Weather.Fog.VisibilityMaximum > 6000 {
		logger.Errorf("fog maximum visibility %f is >6000", config.Options.Weather.Fog.VisibilityMaximum)
		config.Options.Weather.Fog.VisibilityMaximum = 6000
		logger.Warnln("fog maximum visibility defaulted to 6000")
	}

	if config.Options.Weather.Fog.VisibilityMinimum > config.Options.Weather.Fog.VisibilityMaximum {
		logger.Errorf("fog minimum visibility is greater than fog maximum visibility")
		config.Options.Weather.Fog.VisibilityMaximum = 6000
		config.Options.Weather.Fog.VisibilityMinimum = 0
		logger.Warnln("fog minimum visibility defaulted to 0")
		logger.Warnln("fog maximum visibility defaulted to 6000")
	}
}

// checkOptionsDust enforces dust configuration options
func checkOptionsDust() {
	if config.Options.Weather.Dust.VisibilityMinimum < 300 {
		logger.Errorf("dust visibility minimum %f is <300", config.Options.Weather.Dust.VisibilityMinimum)
		config.Options.Weather.Dust.VisibilityMinimum = 300
		logger.Warnln("dust visibility minimum defaulted to 300")
	}

	if config.Options.Weather.Dust.VisibilityMaximum > 3000 {
		logger.Errorf("dust visibility maximum %f is >3000", config.Options.Weather.Dust.VisibilityMaximum)
		config.Options.Weather.Dust.VisibilityMaximum = 3000
		logger.Warnln("dust visibility maximum defaulted to 3000")
	}

	if config.Options.Weather.Dust.VisibilityMinimum > config.Options.Weather.Fog.VisibilityMaximum {
		logger.Errorf("dust minimum visibility is greater than dust maximum visibility")
		config.Options.Weather.Dust.VisibilityMaximum = 3000
		config.Options.Weather.Dust.VisibilityMinimum = 300
		logger.Warnln("dust visibility minimum defaulted to 300")
		logger.Warnln("dust visibility maximum defaulted to 3000")
	}
}
