package weather

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/evogelsa/DCS-real-weather/util"
)

var SelectedPreset string
var SelectedBase int

func GetWeather() (WeatherData, error) {
	// create http client to fetch weather data, timeout after 5 sec
	timeout := time.Duration(5 * time.Second)
	client := http.Client{Timeout: timeout}

	request, err := http.NewRequest(
		"GET",
		"https://api.checkwx.com/metar/"+util.Config.METAR.ICAO+"/decoded",
		nil,
	)
	if err != nil {
		return WeatherData{}, err
	}
	request.Header.Set("X-API-Key", util.Config.APIKey)

	// make api request
	resp, err := client.Do(request)
	if err != nil {
		return WeatherData{}, fmt.Errorf(
			"Error making request to CheckWX: %v",
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
			"Error parsing CheckWX response: %v",
			err,
		)
	}

	log.Println("Received data:", string(body))

	// format json resposne into weatherdata struct
	var res WeatherData
	err = json.Unmarshal(body, &res)
	if err != nil {
		return WeatherData{}, err
	}

	if err := checkWeather(&res); err != nil {
		return res, err
	}

	return res, nil
}

// checkWeather takes in weather data from the API and checks the first results
// for reasonable results that can be applied to DCS weather. In the case of
// bad or missing data, it modifies the value in data to a reasonable default
func checkWeather(data *WeatherData) error {
	if data.NumResults < 1 {
		return fmt.Errorf("no data to check")
	}

	if data.Data[0].Barometer.Hg == 0 {
		log.Println("No barometer data from CheckWX, defaulting to 760 mmHg")
		data.Data[0].Barometer.Hg = 29.92
		data.Data[0].Barometer.HPa = 1013.2
		data.Data[0].Barometer.KPa = 101.32
		data.Data[0].Barometer.MB = 1013.2
	}

	return nil
}

// LogMETAR generates a metar based on the weather settings added to the DCS miz
func LogMETAR(wx WeatherData) error {
	data := wx.Data[0]

	var metar string

	// add ICAO
	metar += "METAR: " + data.ICAO + " "

	// get observed time, no need to translate time zone since it's in Zulu
	t, err := time.Parse("2006-01-02T15:04:05Z", data.Observed)
	if err != nil {
		return fmt.Errorf("Error parsing METAR time: %v", err)
	}
	// want format DDHHMMZ
	metar += fmt.Sprintf("%02d%02d%02dZ ", t.Day(), t.Hour(), t.Minute())

	// winds DIRSPDKT
	metar += fmt.Sprintf(
		"%03d%02dKT ",
		int(data.Wind.Degrees),
		int(data.Wind.SpeedKTS),
	)

	// visibility
	metar += fmt.Sprintf("%dSM ", int(data.Visibility.MilesFloat))

	// conditions
	for _, cond := range data.Conditions {
		metar += fmt.Sprintf("%s ", cond.Code)
	}

	// clouds
	if SelectedPreset == "" {
		metar += "CLR "
	} else {
		clouds := decodePreset[SelectedPreset]
		for i, cld := range clouds {
			if i == 0 {
				// convert base to hundreds of feet
				base := int(float64(SelectedBase)*3.28+50) / 100
				metar += fmt.Sprintf("%s%03d ", cld.Name, base)
			} else {
				metar += fmt.Sprintf("%s%s ", cld.Name, cld.Base)
			}
		}
	}

	// temperature
	if data.Temperature.Celsius < 0 {
		metar += fmt.Sprintf("M%02d/", int(-1*data.Temperature.Celsius))
	} else {
		metar += fmt.Sprintf("%02d/", int(data.Temperature.Celsius))
	}

	// dewpoint
	if data.Dewpoint.Celsius < 0 {
		metar += fmt.Sprintf("M%02d ", int(-1*data.Dewpoint.Celsius))
	} else {
		metar += fmt.Sprintf("%02d ", int(data.Dewpoint.Celsius))
	}

	// altimeter
	metar += fmt.Sprintf("A%4d ", int(data.Barometer.Hg*100))

	// nosig because usually not updated until 4 hours
	metar += "NOSIG "

	// rmks
	metar += util.Config.METAR.Remarks

	log.Println(metar)

	return nil
}

type WeatherData struct {
	Data       []Data `json:"data"`
	NumResults int    `json:"results"`
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

type cloud struct {
	Name string
	Base string
}

var (
	decodePreset = map[string][]cloud{
		`"Preset1"`:      {{"FEW", "070"}},
		`"Preset2"`:      {{"FEW", "080"}, {"SCT", "230"}},
		`"Preset3"`:      {{"SCT", "080"}, {"FEW", "210"}},
		`"Preset4"`:      {{"SCT", "080"}, {"SCT", "240"}},
		`"Preset5"`:      {{"SCT", "140"}, {"FEW", "270"}, {"BKN", "400"}},
		`"Preset6"`:      {{"SCT", "080"}, {"FEW", "400"}},
		`"Preset7"`:      {{"BKN", "075"}, {"SCT", "210"}, {"SCT", "400"}},
		`"Preset8"`:      {{"SCT", "180"}, {"FEW", "360"}, {"FEW", "400"}},
		`"Preset9"`:      {{"BKN", "075"}, {"SCT", "200"}, {"FEW", "410"}},
		`"Preset10"`:     {{"SCT", "180"}, {"FEW", "360"}, {"FEW", "400"}},
		`"Preset11"`:     {{"BKN", "180"}, {"BKN", "320"}, {"FEW", "410"}},
		`"Preset12"`:     {{"BKN", "120"}, {"SCT", "220"}, {"FEW", "410"}},
		`"Preset13"`:     {{"BKN", "120"}, {"BKN", "260"}, {"FEW", "410"}},
		`"Preset14"`:     {{"BKN", "070"}, {"FEW", "410"}},
		`"Preset15"`:     {{"SCT", "140"}, {"BKN", "240"}, {"FEW", "400"}},
		`"Preset16"`:     {{"BKN", "140"}, {"BKN", "280"}, {"FEW", "400"}},
		`"Preset17"`:     {{"BKN", "070"}, {"BKN", "200"}, {"BKN", "320"}},
		`"Preset18"`:     {{"BKN", "130"}, {"BKN", "250"}, {"BKN", "380"}},
		`"Preset19"`:     {{"OVC", "090"}, {"BKN", "230"}, {"BKN", "310"}},
		`"Preset20"`:     {{"BKN", "130"}, {"BKN", "280"}, {"FEW", "380"}},
		`"Preset21"`:     {{"BKN", "070"}, {"OVC", "170"}},
		`"Preset22"`:     {{"OVC", "070"}, {"BKN", "170"}},
		`"Preset23"`:     {{"OVC", "110"}, {"BKN", "180"}, {"SCT", "320"}},
		`"Preset24"`:     {{"OVC", "030"}, {"OVC", "170"}, {"BKN", "340"}},
		`"Preset25"`:     {{"OVC", "120"}, {"OVC", "220"}, {"OVC", "400"}},
		`"Preset26"`:     {{"OVC", "090"}, {"BKN", "230"}, {"SCT", "320"}},
		`"Preset27"`:     {{"OVC", "080"}, {"BKN", "250"}, {"BKN", "340"}},
		`"RainyPreset1"`: {{"OVC", "030"}, {"OVC", "280"}, {"FEW", "400"}},
		`"RainyPreset2"`: {{"OVC", "030"}, {"SCT", "180"}, {"FEW", "400"}},
		`"RainyPreset3"`: {{"OVC", "060"}, {"OVC", "190"}, {"SCT", "340"}},
	}
)

var DefaultWeather WeatherData = WeatherData{
	Data: []Data{
		{
			Wind: Wind{
				SpeedMPS: 1.25,
				Degrees:  270,
				GustMPS:  3,
			},
			Temperature: Temperature{
				Celsius: 15,
			},
			Barometer: Barometer{
				Hg: 29.92,
			},
			Clouds: []Clouds{
				{
					Code: "CLR",
				},
			},
			Observed: time.Now().Format("2006/01/02"),
		},
	},
	NumResults: 1,
}
