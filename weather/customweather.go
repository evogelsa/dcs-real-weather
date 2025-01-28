package weather

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/evogelsa/dcs-real-weather/v2/logger"
)

func getWeatherCustom(filename string) (WeatherData, error) {
	var data WeatherData

	b, err := os.ReadFile(filename)
	if err != nil {
		return data, fmt.Errorf("unable to read custom weather: %v", err)
	}

	var minify bytes.Buffer
	if err := json.Compact(&minify, b); err == nil {
		logger.Infoln("read weather data: %s", minify.String())
	} else {
		logger.Infoln("couldn't minify custom weather data: %v", err)
	}

	if err := json.Unmarshal(b, &data); err != nil {
		return data, fmt.Errorf("could not parse custom weather: %v", err)
	}
	logger.Infoln("parsed custom weather data")

	// if custom weather is provided by rw bot, then remove after done
	if err := os.Remove(".rwbot"); err == nil {
		logger.Infoln("removing custom weather set by bot")
		os.Remove(filename)
	}

	return data, nil
}
