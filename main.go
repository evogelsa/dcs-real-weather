package main

import (
	"github.com/evogelsa/DCS-real-weather/miz"
	"github.com/evogelsa/DCS-real-weather/util"
	"github.com/evogelsa/DCS-real-weather/weather"
)

func main() {
	// parse configuration file
	util.ParseConfig()

	// get METAR report
	data := weather.GetWeather()

	// unzip mission file
	_, err := miz.Unzip()
	util.Must(err)

	// update mission file with weather data
	miz.Update(data)

	// repack mission file contents and form realweather.miz output
	miz.Zip()

	// remove unpacked contents from directory
	miz.Clean()

	weather.LogMETAR(data)
}
