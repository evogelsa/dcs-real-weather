package weather

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/evogelsa/DCS-real-weather/util"
)

func GetWeather() WeatherData {
	// parse config file for parameters
	config := util.ParseConfig()

	// create http client to fetch weather data, timeout after 5 sec
	timeout := time.Duration(5 * time.Second)
	client := http.Client{Timeout: timeout}

	request, err := http.NewRequest(
		"GET",
		"https://api.checkwx.com/metar/"+config.ICAO+"/decoded",
		nil,
	)
	util.Must(err)
	request.Header.Set("X-API-Key", config.APIKey)

	// make api request
	resp, err := client.Do(request)
	util.Must(err)
	defer resp.Body.Close()

	// parse response byte array
	body, err := ioutil.ReadAll(resp.Body)
	util.Must(err)

	fmt.Println(string(body))

	// format json resposne into weatherdata struct
	var res WeatherData
	err = json.Unmarshal(body, &res)
	util.Must(err)

	return res
}

type WeatherData struct {
	Data         []Data `json:"data"`
	WeatherDatas int    `json:"results"`
}

type Data struct {
	Barometer      Barometer      `json:"barometer"`
	Ceiling        Ceiling        `json:"ceiling"`
	Clouds         []Clouds       `json:"clouds"`
	Conditions     []Conditions   `json:"conditions"`
	Dewpoint       Dewpoint       `json:"dewpoint"`
	Elevation      Elevation      `json:"elevation"`
	FlightCategory FlightCategory `json:"flight_category"`
	Humidity       Humidity       `json:"humidity"`
	ICAO           string         `json:"icao"`
	ID             string         `json:"id"`
	Location       Location       `json:"location"`
	Observed       string         `json:"observed"`
	RawText        string         `json:"raw_text"`
	Station        Station        `json:"station"`
	Temperature    Temperature    `json:"temperature"`
	Visibility     Visibility     `json:"visibility"`
	Wind           Wind           `json:"wind"`
}

type Barometer struct {
	Hg  float32 `json:"hg"`
	HPa float32 `json:"hpa"`
	KPa float32 `json:"kpa"`
	MB  float32 `json:"mb"`
}

type Ceiling struct {
	BaseFeetAGL   float32 `json:"base_feet_agl"`
	BaseMetersAGL float32 `json:"base_meters_agl"`
	Code          string  `json:"code"`
	Feet          float32 `json:"feet"`
	Meters        float32 `json:"meters"`
	Text          string  `json:"text"`
}

type Clouds struct {
	BaseFeetAGL   float32 `json:"base_feet_agl"`
	BaseMetersAGL float32 `json:"base_meters_agl"`
	Code          string  `json:"code"`
	Feet          float32 `json:"feet"`
	Meters        float32 `json:"meters"`
	Text          string  `json:"text"`
}

type Conditions struct {
	Code string `json:"code"`
	Text string `json:"text"`
}

type Dewpoint struct {
	Celsius    float32 `json:"celsius"`
	Fahrenheit float32 `json:"fahrenheit"`
}

type Elevation struct {
	Feet   float32 `json:"feet"`
	Meters float32 `json:"meters"`
}

type FlightCategory string

type Humidity struct {
	Percent float32 `json:"percent"`
}

type Location struct {
	Coordinates []float32 `json:"coordinates"`
	Type        string    `json:"type"`
}

type Station struct {
	Name string `json:"name"`
}

type Temperature struct {
	Celsius    float32 `json:"celsius"`
	Fahrenheit float32 `json:"fahrenheit"`
}

type Visibility struct {
	Meters      string  `json:"meters"`
	MetersFloat float32 `json:"meters_float"`
	Miles       string  `json:"miles"`
	MilesFloat  float32 `json:"miles_float"`
}

type Wind struct {
	Degrees  float32 `json:"degrees"`
	SpeedKPH float32 `json:"speed_kph"`
	SpeedKTS float32 `json:"speed_kts"`
	SpeedMPH float32 `json:"speed_mph"`
	SpeedMPS float32 `json:"speed_mps"`
	GustKPH  float32 `json:"gust_kph"`
	GustKTS  float32 `json:"gust_kts"`
	GustMPH  float32 `json:"gust_mph"`
	GustMPS  float32 `json:"gust_mps"`
}
