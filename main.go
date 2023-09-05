package main

//go:generate goversioninfo versioninfo/versioninfo.json

import (
	"log"

	"github.com/evogelsa/DCS-real-weather/miz"
	"github.com/evogelsa/DCS-real-weather/util"
	"github.com/evogelsa/DCS-real-weather/weather"
)

func main() {
	// parse configuration file
	util.ParseConfig()

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

	// sanity check data before updating mission
	if data.WeatherDatas > 0 && data.Data[0].Barometer.Hg > 0 {
		// update mission file with weather data
		miz.Update(data)
	} else {
		log.Println("Incorrect weather data. No real weather applied to mission file.")
	}

	// repack mission file contents and form realweather.miz output
	miz.Zip()

	// remove unpacked contents from directory
	miz.Clean()

	if data.WeatherDatas > 0 {
		weather.LogMETAR(data)
	}
}
