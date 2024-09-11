package main

//go:generate goversioninfo -o resource.syso ../../versioninfo/versioninfo.json

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"path/filepath"
	"runtime"

	"github.com/evogelsa/DCS-real-weather/config"
	"github.com/evogelsa/DCS-real-weather/miz"
	"github.com/evogelsa/DCS-real-weather/versioninfo"
	"github.com/evogelsa/DCS-real-weather/weather"
)

// flag vars
var (
	debugCheckWx bool
)

func init() {
	flag.BoolVar(&debugCheckWx, "debug-checkwx", false, "load checkwx data from checkwx.json")

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

	log.Println("Using Real Weather " + ver)
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
	if err = miz.Update(&data, windsAloft); err != nil {
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

	// use value from config if exists. CLI argument will override a false
	// parameter in the config
	if config.Get().API.Custom.Enable {
		debugCheckWx = true
	}

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
		var meta string
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
		}

		if err == nil {
			break
		} else {
			log.Printf("Error getting weather from %s: %v", api.Provider, err)
		}
	}

	if err != nil {
		log.Printf("Error getting weather, using default: %v\n", err)
		data = weather.DefaultWeather
	}

	// override with custom weather if enabled
	overrideWx(icao, &data)

	return data
}

// overrideWx handles overriding weather if enabled
func overrideWx(icao string, data *weather.WeatherData) {
	if !config.Get().API.Custom.Enable || !config.Get().API.Custom.Override {
		return
	}

	log.Println("Overriding weather with custom data from file...")

	temp, err := weather.GetWeather(icao, weather.APICustom, config.Get().API.Custom.File)
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
