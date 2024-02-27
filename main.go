package main

//go:generate goversioninfo versioninfo/versioninfo.json

import (
	"fmt"
	"log"

	"github.com/evogelsa/DCS-real-weather/miz"
	"github.com/evogelsa/DCS-real-weather/versioninfo"
	"github.com/evogelsa/DCS-real-weather/weather"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Unexpected error encountered: %v\n", r)
		}
	}()

	// log version
	var ver string

	ver += fmt.Sprintf("v%d.%d.%d", versioninfo.Major, versioninfo.Minor, versioninfo.Patch)
	if versioninfo.Pre != "" {
		ver += fmt.Sprintf("-%s%d", versioninfo.Pre, versioninfo.CommitNum)
	}
	if versioninfo.Commit != "" {
		ver += "+" + versioninfo.Commit
	}

	log.Println("Using Real Weather " + ver)

	// get METAR report
	var err error
	var data weather.WeatherData
	data, err = weather.GetWeather()
	if err != nil {
		log.Printf("Error getting weather, using default: %v\n", err)
		data = weather.DefaultWeather
	}

	// unzip mission file
	_, err = miz.Unzip()
	if err != nil {
		log.Fatalf("Error unzipping mission file: %v\n", err)
	}

	// confirm there is data before updating
	if data.NumResults > 0 {
		// update mission file with weather data
		if err := miz.Update(data); err != nil {
			log.Printf("Error updating mission: %v\n", err)
		}
	} else {
		log.Println("Incorrect weather data. No real weather applied to mission file.")
	}

	// repack mission file contents and form realweather.miz output
	if err := miz.Zip(); err != nil {
		log.Fatalf("Error repacking mission file: %v\n", err)
	}

	// remove unpacked contents from directory
	// miz.Clean()

	if data.NumResults > 0 {
		if err := weather.LogMETAR(data); err != nil {
			log.Printf("Could not create DCS METAR: %v", err)
		}
	}
}
