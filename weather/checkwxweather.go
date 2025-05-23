package weather

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/evogelsa/DCS-real-weather/v2/logger"
)

type WeatherData struct {
	Data       []Data `json:"data,omitempty"`
	NumResults int    `json:"results,omitempty"`
}

type Data struct {
	Barometer      *Barometer   `json:"barometer,omitempty"`
	Clouds         []Clouds     `json:"clouds,omitempty"`
	Conditions     []Conditions `json:"conditions,omitempty"`
	Dewpoint       *Dewpoint    `json:"dewpoint,omitempty"`
	FlightCategory string       `json:"flight_category,omitempty"`
	ICAO           string       `json:"icao,omitempty"`
	ID             string       `json:"id,omitempty"`
	Observed       string       `json:"observed,omitempty"`
	RawText        string       `json:"raw_text,omitempty"`
	Station        *Station     `json:"station,omitempty"`
	Temperature    *Temperature `json:"temperature,omitempty"`
	Visibility     *Visibility  `json:"visibility,omitempty"`
	Wind           *Wind        `json:"wind,omitempty"`
}

type Barometer struct {
	Hg float64 `json:"hg,omitempty"`
	// HPa float64 `json:"hpa,omitempty"`
	// KPa float64 `json:"kpa,omitempty"`
	// MB  float64 `json:"mb,omitempty"`
}

type Clouds struct {
	// BaseFeetAGL   float64 `json:"base_feet_agl,omitempty"`
	// BaseMetersAGL float64 `json:"base_meters_agl,omitempty"`
	Code string `json:"code,omitempty"`
	// Feet          float64 `json:"feet,omitempty"`
	Meters float64 `json:"meters,omitempty"`
	// Text          string  `json:"text,omitempty"`
}

type Conditions struct {
	Code string `json:"code,omitempty"`
	// Text string `json:"text,omitempty"`
}

type Dewpoint struct {
	Celsius float64 `json:"celsius,omitempty"`
	// Fahrenheit float64 `json:"fahrenheit,omitempty"`
}

type Station struct {
	// Location string    `json:"location,omitempty"`
	// Name     string    `json:"name,omitempty"`
	// Type     string    `json:"type,omitempty"`
	Geometry *Geometry `json:"geometry,omitempty"`
}

type Geometry struct {
	Coordinates []float64 `json:"coordinates,omitempty"` // longitude, latitude
	// Type        string    `json:"type,omitempty"`
}

type Temperature struct {
	Celsius float64 `json:"celsius,omitempty"`
	// Fahrenheit float64 `json:"fahrenheit,omitempty"`
}

type Visibility struct {
	// Meters      string  `json:"meters_text,omitempty"`
	MetersFloat float64 `json:"meters,omitempty"`
	// Miles       string  `json:"miles_text,omitempty"`
	// MilesFloat  float64 `json:"miles,omitempty"`
}

type Wind struct {
	Degrees  float64 `json:"degrees,omitempty"`
	SpeedMPS float64 `json:"speed_mps,omitempty"`
	// GustKPH  float64 `json:"gust_kph,omitempty"`
	// GustKTS  float64 `json:"gust_kts,omitempty"`
	// GustMPH  float64 `json:"gust_mph,omitempty"`
	GustMPS float64 `json:"gust_mps,omitempty"`
}

func getWeatherCheckWX(icao, apiKey string) (WeatherData, error) {
	logger.Infoln("getting weather from CheckWX...")

	// create http client to fetch weather data, timeout after 5 sec
	timeout := time.Duration(5 * time.Second)
	client := http.Client{Timeout: timeout}

	request, err := http.NewRequest(
		"GET",
		"https://api.checkwx.com/metar/"+icao+"/decoded",
		nil,
	)
	if err != nil {
		return WeatherData{}, err
	}
	request.Header.Set("X-API-Key", apiKey)

	// make api request
	resp, err := client.Do(request)
	if err != nil {
		return WeatherData{}, fmt.Errorf(
			"error making request to CheckWX: %v",
			err,
		)
	}

	// verify response OK
	if resp.StatusCode != http.StatusOK {
		return WeatherData{}, fmt.Errorf("CheckWX bad status: %v", resp.Status)
	}
	defer resp.Body.Close()

	// parse response byte array
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return WeatherData{}, fmt.Errorf(
			"error parsing CheckWX response: %v",
			err,
		)
	}

	logger.Infoln("got weather data:", string(body))
	logger.Infoln("parsing weather...")

	// format json resposne into weatherdata struct
	var res WeatherData
	err = json.Unmarshal(body, &res)
	if err != nil {
		return WeatherData{}, err
	}

	logger.Infoln("parsed weather")

	return res, nil
}
