package main

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
	data := weather.GetWeather()

	// unzip mission file
	_, err := miz.Unzip()
	util.Must(err)

	if data.WeatherDatas > 0 {
		if int(data.Data[0].Barometer.Hg*25.4 + .5) > 0 {
			// update mission file with weather data
			miz.Update(data)
		} else {
			log.Println("Incorrect weather data. No real weather applied to mission file.")
		}
	}
	
	// repack mission file contents and form realweather.miz output
	miz.Zip()

	// remove unpacked contents from directory
	miz.Clean()

	if data.WeatherDatas > 0 {
		weather.LogMETAR(data) 
	}
}
