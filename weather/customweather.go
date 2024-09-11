package weather

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func getWeatherCustom(filename string) (WeatherData, error) {
	var data WeatherData

	b, err := os.ReadFile(filename)
	if err != nil {
		return data, fmt.Errorf("Unable to read custom weather: %v", err)
	}

	var minify bytes.Buffer
	if err := json.Compact(&minify, b); err == nil {
		log.Printf("Read weather data: %s", minify.String())
	} else {
		log.Printf("Couldn't minify custom weather data: %v", err)
	}

	if err := json.Unmarshal(b, &data); err != nil {
		return data, fmt.Errorf("Could not parse custom weather: %v", err)
	}
	log.Println("Parsed custom weather data")

	if err := ValidateWeather(&data); err != nil {
		return data, fmt.Errorf("Error validating weather data: %v", err)
	}

	if err := os.Remove(".rwbot"); err == nil {
		log.Println("Removing custom weather set by bot")
		os.Remove(filename)
	}

	return data, nil
}
