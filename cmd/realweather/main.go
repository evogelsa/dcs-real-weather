package main

//go:generate go run ../../versioninfo/generate/generate.go ../../versioninfo/
//go:generate goversioninfo -o resource.syso ../../versioninfo/versioninfo.json

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
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
	if metar, err = weather.GenerateMETAR(data, config.Get().METAR.Remarks); err == nil {
		// make metar last thing to be print
		defer log.Println(metar)
	} else {
		log.Printf("Error creating DCS METAR: %v", err)
	}

	// add METAR to mission brief if enabled
	if config.Get().METAR.AddToBrief {
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
	if config.Get().METAR.UseCustomData {
		debugCheckWx = true
	}

	data, err = weather.GetWeather(config.Get().METAR.ICAO, config.Get().APIKey)
	if err != nil {
		log.Printf("Error getting weather, using default: %v\n", err)
		data = weather.DefaultWeather
	}

	if debugCheckWx {
		log.Println("Overriding weather with custom data from file...")
		b, err := os.ReadFile("checkwx.json")
		if err != nil {
			log.Printf("Could not read custom override weather, no overrides applied")
			return data
		}

		var minify bytes.Buffer
		if err := json.Compact(&minify, b); err == nil {
			log.Println("Read weather data: ", minify.String())
		} else {
			log.Println("Couldn't minify custom weather data:", err)
		}

		err = json.Unmarshal(b, &data)
		if err != nil {
			log.Fatalf("Could not parse checkwx.json: %v", err)
		}
		log.Println("Parsed weather data")

		if err := weather.ValidateWeather(&data); err != nil {
			log.Fatalf("Error validating weather data: %v", err)
		}

		if err := os.Remove(".rwbot"); err == nil {
			log.Println("Removing custom weather set by bot")
			os.Remove("checkwx.json")
		}
	}

	return data
}
