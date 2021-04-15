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

	fmt.Println("Received data:", string(body))

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
	Hg  float64 `json:"hg"`
	HPa float64 `json:"hpa"`
	KPa float64 `json:"kpa"`
	MB  float64 `json:"mb"`
}

type Ceiling struct {
	BaseFeetAGL   float64 `json:"base_feet_agl"`
	BaseMetersAGL float64 `json:"base_meters_agl"`
	Code          string  `json:"code"`
	Feet          float64 `json:"feet"`
	Meters        float64 `json:"meters"`
	Text          string  `json:"text"`
}

type Clouds struct {
	BaseFeetAGL   float64 `json:"base_feet_agl"`
	BaseMetersAGL float64 `json:"base_meters_agl"`
	Code          string  `json:"code"`
	Feet          float64 `json:"feet"`
	Meters        float64 `json:"meters"`
	Text          string  `json:"text"`
}

type Conditions struct {
	Code string `json:"code"`
	Text string `json:"text"`
}

type Dewpoint struct {
	Celsius    float64 `json:"celsius"`
	Fahrenheit float64 `json:"fahrenheit"`
}

type Elevation struct {
	Feet   float64 `json:"feet"`
	Meters float64 `json:"meters"`
}

type FlightCategory string

type Humidity struct {
	Percent float64 `json:"percent"`
}

type Location struct {
	Coordinates []float64 `json:"coordinates"`
	Type        string    `json:"type"`
}

type Station struct {
	Name string `json:"name"`
}

type Temperature struct {
	Celsius    float64 `json:"celsius"`
	Fahrenheit float64 `json:"fahrenheit"`
}

type Visibility struct {
	Meters      string  `json:"meters"`
	MetersFloat float64 `json:"meters_float"`
	Miles       string  `json:"miles"`
	MilesFloat  float64 `json:"miles_float"`
}

type Wind struct {
	Degrees  float64 `json:"degrees"`
	SpeedKPH float64 `json:"speed_kph"`
	SpeedKTS float64 `json:"speed_kts"`
	SpeedMPH float64 `json:"speed_mph"`
	SpeedMPS float64 `json:"speed_mps"`
	GustKPH  float64 `json:"gust_kph"`
	GustKTS  float64 `json:"gust_kts"`
	GustMPH  float64 `json:"gust_mph"`
	GustMPS  float64 `json:"gust_mps"`
}

type CloudPreset struct {
	Name    string
	MinBase int
	MaxBase int
}

var CloudPresets map[string][]CloudPreset = map[string][]CloudPreset{
	"FEW": {
		{`"Preset1"`, 840, 4200},  // Light Scattered 1
		{`"Preset2"`, 1260, 2520}, // Light Scattered 2
	},
	"SCT": {
		{`"Preset3"`, 840, 2520},   // High Scattered 1
		{`"Preset4"`, 1260, 2520},  // High Scattered 2
		{`"Preset5"`, 1260, 4620},  // Scattered 1
		{`"Preset6"`, 1260, 4200},  // Scattered 2
		{`"Preset7"`, 1680, 5040},  // Scattered 3
		{`"Preset8"`, 3780, 5460},  // High Scattered 3
		{`"Preset9"`, 1680, 3780},  // Scattered 4
		{`"Preset10"`, 1260, 4200}, // Scattered 5
		{`"Preset11"`, 2520, 5460}, // Scattered 6
		{`"Preset12"`, 1680, 3360}, // Scattered 7
	},
	"BKN": {
		{`"Preset13"`, 1680, 3360}, // Broken 1
		{`"Preset14"`, 1680, 3360}, // Broken 2
		{`"Preset15"`, 840, 5040},  // Broken 3
		{`"Preset16"`, 1260, 4200}, // Broken 4
		{`"Preset17"`, 0, 2520},    // Broken 5
		{`"Preset18"`, 0, 3780},    // Broken 6
		{`"Preset19"`, 0, 2940},    // Broken 7
		{`"Preset20"`, 0, 3780},    // Broken 8
	},
	"OVC": {
		{`"Preset21"`, 1260, 4200}, // Overcast 1
		{`"Preset22"`, 420, 4200},  // Overcast 2
		{`"Preset23"`, 840, 3360},  // Overcast 3
		{`"Preset24"`, 420, 2520},  // Overcast 4
		{`"Preset25"`, 420, 3360},  // Overcast 5
		{`"Preset26"`, 420, 2940},  // Overcast 6
		{`"Preset27"`, 420, 2520},  // Overcast 7
	},
	"OVC+RA": {
		{`"RainyPreset1"`, 420, 2940}, // Overcast And Rain 1
		{`"RainyPreset2"`, 840, 2520}, // Overcast And Rain 2
		{`"RainyPreset3"`, 840, 2520}, // Overcast And Rain 3
	},
}
