package main

//go:generate goversioninfo -o resource.syso ../../versioninfo/versioninfo.json

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"

	"go.uber.org/zap/zapcore"

	"github.com/evogelsa/DCS-real-weather/v2/config"
	"github.com/evogelsa/DCS-real-weather/v2/logger"
	"github.com/evogelsa/DCS-real-weather/v2/miz"
	"github.com/evogelsa/DCS-real-weather/v2/versioninfo"
	"github.com/evogelsa/DCS-real-weather/v2/weather"
)

// flag vars
var (
	enableCustom bool
	validate     bool
	version      bool

	configName    string
	customFile    string
	icao          string
	inputMission  string
	outputMission string
)

func init() {
	const usage = `Usage of %s:
	Boolean Flags:
		-enable-custom  forcibly enable the custom weather provider
		-help           prints this help message
		-validate       validates your config the exits
		-version        prints the Real Weather version then exits

	String Flags:
		-config         override default config file name
		-custom-file    override file path for custom weather provider
		-icao           override icao
		-input          override input mission
		-output         override output mission
`

	flag.Usage = func() {
		fmt.Fprintf(
			flag.CommandLine.Output(),
			usage,
			os.Args[0],
		)
	}

	flag.BoolVar(&enableCustom, "enable-custom", false, "forcibly enables the custom weather provider")
	flag.BoolVar(&validate, "validate", false, "validates your config then exits")
	flag.BoolVar(&version, "version", false, "prints out the real weather version and exits")

	flag.StringVar(&configName, "config", "config.toml", "override default config file name")
	flag.StringVar(&customFile, "custom-file", "", "override file path for custom weather provider")
	flag.StringVar(&icao, "icao", "", "override icao in config")
	flag.StringVar(&inputMission, "input", "", "override input mission in config")
	flag.StringVar(&outputMission, "output", "", "override output mission in config")

	flag.Parse()
}

func init() {
	// if .rwbot file exists, then override custom provider for this run only
	// .rwbot files will be cleaned up (deleted) after using custom data
	// provider
	if _, err := os.Stat(".rwbot"); !errors.Is(err, os.ErrNotExist) {
		enableCustom = true
		customFile = ".rwbotwx.json"
	}

	// set config overrides
	overrides := config.Overrideable{
		APICustomEnable:    enableCustom,
		APICustomFile:      customFile,
		MissionInput:       inputMission,
		MissionOutput:      outputMission,
		OptionsWeatherICAO: icao,
	}

	config.Init(configName, overrides)

	var logfile string
	if config.Get().RealWeather.Log.Enable {
		logfile = config.Get().RealWeather.Log.File
	}

	var level zapcore.Level
	switch config.Get().RealWeather.Log.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	logger.Init(
		logfile,
		config.Get().RealWeather.Log.MaxSize,
		config.Get().RealWeather.Log.MaxBackups,
		config.Get().RealWeather.Log.MaxAge,
		config.Get().RealWeather.Log.Compress,
		level,
	)

	// log version
	var ver string

	ver += fmt.Sprintf("v%d.%d.%d", versioninfo.Major, versioninfo.Minor, versioninfo.Patch)
	if versioninfo.Pre != "" {
		ver += fmt.Sprintf("-%s-%d", versioninfo.Pre, versioninfo.CommitNum)
	}

	if versioninfo.Commit != "" {
		ver += "+" + versioninfo.Commit
	}

	logger.Infof("using real weather %s", ver)
	if version {
		os.Exit(0)
	}

	config.Validate()

	if validate {
		os.Exit(0)
	}
}

func main() {
	var err error

	defer func() {
		if r := recover(); r != nil {
			_, fn, line, ok := runtime.Caller(4)
			if ok {
				base := filepath.Base(fn)
				e := fmt.Sprintf("%s:%d", base, line)
				logger.Fatalf("unexpected error encountered: %s %v", e, r)
			} else {
				logger.Fatalf("unexpected error encountered: %v", r)
			}
		}
	}()

	data := getWx()

	// confirm there is data before updating
	if data.NumResults <= 0 {
		logger.Fatalf("no weather data received")
	}

	// get winds aloft
	var windsAloft weather.WindsAloft
	if config.Get().API.OpenMeteo.Enable {
		windsAloft, err = weather.GetWindsAloft(data.Data[0].Station.Geometry.Coordinates)
		if err != nil {
			logger.Errorf("error getting winds aloft: %v", err)
			config.Set("open-meteo", false)
			logger.Warnln("continuing with legacy winds")
		}
	}

	// unzip mission file
	_, err = miz.Unzip()
	if err != nil {
		logger.Fatalf("error unpacking mission file: %v\n", err)
	}

	// update mission file with weather data
	if err = miz.UpdateMission(&data, windsAloft); err != nil {
		logger.Errorf("error updating mission: %v\n", err)
	}

	// generate the METAR text
	var metar string
	if metar, err = weather.GenerateMETAR(data, config.Get().RealWeather.Mission.Brief.Remarks); err == nil {
		// make metar last thing to be print
		defer logger.Infof("METAR: %s", metar)
	} else {
		logger.Errorf("error creating METAR: %v", err)
	}

	// add METAR to mission brief if enabled
	if config.Get().RealWeather.Mission.Brief.AddMETAR {
		if err = miz.UpdateBrief(metar); err != nil {
			logger.Errorf("error adding METAR to brief: %v", err)
		}
	}

	// repack mission file contents and form realweather.miz output
	if err := miz.Zip(); err != nil {
		logger.Fatalf("error repacking mission file: %v", err)
	}

	// remove unpacked contents from directory
	miz.Clean()
}

func getWx() weather.WeatherData {
	// get METAR report
	var err error
	var data weather.WeatherData

	var icao string
	if config.Get().Options.Weather.ICAO != "" {
		icao = config.Get().Options.Weather.ICAO
	} else if len(config.Get().Options.Weather.ICAOList) > 0 {
		icao = config.Get().Options.Weather.ICAOList[rand.Intn(len(config.Get().Options.Weather.ICAOList))]
	} else {
		// Should never reach this code if config validation is working properly
		logger.Errorf("icao config validation failed, please report this as a bug :-)")
		icao = "UGKO"
		logger.Warnln("icao defaulted to UGKO")
	}

	// construct usable api priority list from config
	apiList := make([]struct {
		Provider weather.API
		Enable   bool
	}, len(config.Get().API.ProviderPriority))
	for i, provider := range config.Get().API.ProviderPriority {
		switch weather.API(provider) {
		case weather.APICheckWX:
			apiList[i].Provider = weather.APICheckWX
			apiList[i].Enable = config.Get().API.CheckWX.Enable

		case weather.APIAviationWeather:
			apiList[i].Provider = weather.APIAviationWeather
			apiList[i].Enable = config.Get().API.AviationWeather.Enable

		case weather.APICustom:
			apiList[i].Provider = weather.APICustom
			apiList[i].Enable = config.Get().API.Custom.Enable

		default:
			logger.Warnln("ignoring unrecognized provider \"%s\"", provider)
		}
	}

	// use first enabled api that works (based on priority list)
	for _, api := range apiList {
		var meta interface{}
		switch api.Provider {
		case weather.APIAviationWeather:
			meta = ""
		case weather.APICheckWX:
			meta = config.Get().API.CheckWX.Key
		case weather.APICustom:
			meta = config.Get().API.Custom.File
		}

		if api.Enable {
			data, err = weather.GetWeather(icao, api.Provider, meta)

			if err == nil {
				break
			} else {
				logger.Errorf("error getting weather from %s: %v", api.Provider, err)
			}
		}

	}

	if err != nil {
		logger.Errorf("could not get any weather data") // don't reprint error
		data = weather.DefaultWeather
		logger.Warnln("using default weather")
	}

	// override with custom weather if enabled
	overrideWx(icao, &data)

	if err := weather.ValidateWeather(&data); err != nil {
		logger.Errorf("error validating weather: %v", err)
	}

	return data
}

// overrideWx handles overriding weather if enabled
func overrideWx(icao string, data *weather.WeatherData) {
	if !config.Get().API.Custom.Enable || !config.Get().API.Custom.Override {
		return
	}

	logger.Infoln("overriding weather with custom data from file...")

	meta := config.Get().API.Custom.File

	temp, err := weather.GetWeather(icao, weather.APICustom, meta)
	if err != nil {
		logger.Errorf("unable to get custom weather: %v", err)
		return
	}

	marshalled, err := json.Marshal(temp)
	if err != nil {
		logger.Errorf("unable to marshal custom weather: %v", err)
		return
	}

	if err := json.Unmarshal(marshalled, data); err != nil {
		logger.Errorf("unable to unmarshal overrides: %v", err)
		return
	}

	logger.Infoln("weather overrides applied")
}
