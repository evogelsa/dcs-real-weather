package main

//go:generate goversioninfo -o resource.syso ../../versioninfo/versioninfo.json

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"

	"github.com/evogelsa/DCS-real-weather/config"
	"github.com/evogelsa/DCS-real-weather/miz"
	"github.com/evogelsa/DCS-real-weather/versioninfo"
	"github.com/evogelsa/DCS-real-weather/weather"
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
	// log version
	var ver string

	ver += fmt.Sprintf("v%d.%d.%d", versioninfo.Major, versioninfo.Minor, versioninfo.Patch)
	if versioninfo.Pre != "" {
		ver += fmt.Sprintf("-%s-%d", versioninfo.Pre, versioninfo.CommitNum)
	}

	if versioninfo.Commit != "" {
		ver += "+" + versioninfo.Commit
	}

	defer log.Println("Using Real Weather " + ver)
	if version {
		os.Exit(0)
	}

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
				ext := filepath.Ext(fn)
				trim := base[:len(base)-len(ext)]
				e := fmt.Sprintf("%s:%d", trim, line)
				log.Printf("Unexpected error encountered: %s %v", e, r)
			} else {
				log.Printf("Unexpected error encountered: %v", r)
			}
		}
	}()

	data := getWx()

	// confirm there is data before updating
	if data.NumResults <= 0 {
		log.Fatalf("Incorrect weather data. No weather applied to mission file.")
	}

	// get winds aloft
	var windsAloft weather.WindsAloft
	windsAloft, err = weather.GetWindsAloft(data.Data[0].Station.Geometry.Coordinates)
	if err != nil {
		log.Printf("Error getting winds aloft, using legacy winds aloft: %v", err)
		config.Set("open-meteo", false)
	}

	// unzip mission file
	_, err = miz.Unzip()
	if err != nil {
		log.Fatalf("Error unzipping mission file: %v\n", err)
	}

	// update mission file with weather data
	if err = miz.UpdateMission(&data, windsAloft); err != nil {
		log.Printf("Error updating mission: %v\n", err)
	}

	// generate the METAR text
	var metar string
	if metar, err = weather.GenerateMETAR(data, config.Get().RealWeather.Mission.Brief.Remarks); err == nil {
		// make metar last thing to be print
		defer log.Println("METAR: " + metar)
	} else {
		log.Printf("Error creating DCS METAR: %v", err)
	}

	// add METAR to mission brief if enabled
	if config.Get().RealWeather.Mission.Brief.AddMETAR {
		if err = miz.UpdateBrief(metar); err != nil {
			log.Printf("Error adding METAR to brief: %v", err)
		}
	}

	// repack mission file contents and form realweather.miz output
	if err := miz.Zip(); err != nil {
		log.Fatalf("Error repacking mission file: %v", err)
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
		log.Println("ICAO config validation error, using DGAA. Please report this as a bug :-)")
		icao = "DGAA"
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
			log.Printf("Unrecognized API provider %s, ignoring", provider)
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
				log.Printf("Error getting weather from %s: %v", api.Provider, err)
			}
		}

	}

	if err != nil {
		log.Println("Error getting weather, using default")
		data = weather.DefaultWeather
	}

	// override with custom weather if enabled
	overrideWx(icao, &data)

	if err := weather.ValidateWeather(&data); err != nil {
		log.Printf("Error validating weather: %v", err)
	}

	return data
}

// overrideWx handles overriding weather if enabled
func overrideWx(icao string, data *weather.WeatherData) {
	if !config.Get().API.Custom.Enable || !config.Get().API.Custom.Override {
		return
	}

	log.Println("Overriding weather with custom data from file...")

	meta := config.Get().API.Custom.File

	temp, err := weather.GetWeather(icao, weather.APICustom, meta)
	if err != nil {
		log.Printf("Unable to get custom weather: %v", err)
		return
	}

	marshalled, err := json.Marshal(temp)
	if err != nil {
		log.Printf("Unable to marshal custom weather: %v", err)
		return
	}

	if err := json.Unmarshal(marshalled, data); err != nil {
		log.Printf("Unable to unmarshal overrides: %v", err)
		return
	}

	log.Println("Weather overrides applied")
}
