package weather

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"time"
)

var (
	SelectedPreset string
	SelectedBase   int
)

const (
	MPSToKT      = 1.944
	KtToMPS      = 0.5144
	MetersToFeet = 3.281
	FeetToMeters = 0.3048
	HPaToInHg    = 0.02953
	InHgToHPa    = 33.86
	HPaPerMeter  = 0.111
	InHgToMMHg   = 25.4
)

const (
	degToRad = math.Pi / 180
)

func ClearCodes() []string {
	return []string{"CAVOK", "CLR", "SKC", "NSC", "NCD"}
}

func CelsiusToFahrenheit(c float64) float64 {
	return (c * 1.8) + 32
}

func FahrenheitToCelsius(f float64) float64 {
	return (f - 32) / 1.8
}

// QNHToQFF takes a QNH value in hPa, elevation in meters, temperature in
// Celsius, and latitude in degrees and returns the equivalent QFF values
func QNHToQFF(qnh, elevation, temperature, latitude float64) float64 {
	qfe := qnh - HPaPerMeter*elevation
	var t float64

	// handle inversions using SMHI method
	if temperature < -7 {
		t = 0.5*temperature + 275
	} else if temperature < 2 {
		t = 0.535*temperature + 275.6
	} else {
		t = 1.07*temperature + 274.5
	}

	qff := qfe * math.Pow(math.E, (elevation*0.034163*(1-0.0026373*math.Cos(latitude*degToRad)))/t)
	return qff
}

func GetWindsAloft(location []float64) (WindsAloft, error) {
	log.Println("Getting winds aloft data from Open Meteo...")

	// create http client to fetch weather data, timeout after 5 sec
	timeout := time.Duration(5 * time.Second)
	client := http.Client{Timeout: timeout}

	request, err := http.NewRequest(
		"GET",
		"https://api.open-meteo.com/v1/forecast",
		nil,
	)
	if err != nil {
		return WindsAloft{}, err
	}

	// add query parameters
	q := request.URL.Query()
	q.Add("latitude", fmt.Sprintf("%.6f", location[1]))
	q.Add("longitude", fmt.Sprintf("%.6f", location[0]))
	q.Add("hourly", "windspeed_800hPa,windspeed_400hPa,winddirection_800hPa,winddirection_400hPa")
	q.Add("wind_speed_unit", "ms")

	request.URL.RawQuery = q.Encode()

	// make request
	resp, err := client.Do(request)
	if err != nil {
		return WindsAloft{}, err
	}

	// verify response
	if resp.StatusCode != http.StatusOK {
		return WindsAloft{}, fmt.Errorf("Open Meteo bad status: %v", resp.Status)
	}

	// parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return WindsAloft{}, fmt.Errorf("Error parsing Open Meteo response: %v", err)
	}

	log.Println("Got winds aloft data")
	log.Println("Parsing winds aloft data...")

	// format response into winds aloft struct
	var res OpenMeteoData
	err = json.Unmarshal(body, &res)
	if err != nil {
		return WindsAloft{}, err
	}

	// get current time
	t := time.Now().UTC().Format("2006-01-02T15") + ":00"

	// find index of current timestamp
	var i int
	var ts string
	for i, ts = range res.Hourly.Time {
		if t == ts {
			break
		}
	}

	// create return windspeed and winddir arrays
	data := WindsAloft{
		WindSpeed1900:     res.Hourly.WindSpeed1900[i],
		WindSpeed7200:     res.Hourly.WindSpeed7200[i],
		WindDirection1900: res.Hourly.WindDirection1900[i],
		WindDirection7200: res.Hourly.WindDirection7200[i],
	}

	log.Printf("Parsed winds aloft data: %v", data)

	return data, nil
}

func GetWeather(icao, apiKey string) (WeatherData, error) {
	log.Println("Getting weather from CheckWX...")

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

	log.Println("Got weather data:", string(body))
	log.Println("Parsing weather...")

	// format json resposne into weatherdata struct
	var res WeatherData
	err = json.Unmarshal(body, &res)
	if err != nil {
		return WeatherData{}, err
	}

	if err := ValidateWeather(&res); err != nil {
		return res, err
	}

	log.Println("Parsed weather")

	return res, nil
}

// ValidateWeather takes in weather data from the API and checks the first
// results for reasonable results that can be applied to DCS weather. In the
// case of bad or missing data, it modifies the value in data to a reasonable
// default
func ValidateWeather(data *WeatherData) error {
	if data.NumResults < 1 {
		return fmt.Errorf("no data to check")
	}

	if data.Data[0].Barometer == nil {
		log.Println("No barometer data, defaulting to 760 mmHg")
		data.Data[0].Barometer = &Barometer{
			Hg:  29.92,
			HPa: 1013.2,
			KPa: 101.32,
			MB:  1013.2,
		}
	}

	if data.Data[0].Ceiling == nil {
		log.Println("No ceiling data, defaulting to clear")
		data.Data[0].Ceiling = &Ceiling{
			BaseFeetAGL:   0,
			BaseMetersAGL: 0,
			Code:          "CLR",
			Feet:          0,
			Meters:        0,
			Text:          "Clear",
		}
	}

	if data.Data[0].Dewpoint == nil {
		log.Println("No dewpoint data, defaulting to 0 Celsius")
		data.Data[0].Dewpoint = &Dewpoint{
			Celsius:    0,
			Fahrenheit: 32,
		}
	}

	if data.Data[0].Elevation == nil {
		log.Println("No elevation data, defaulting to 0 meters")
		data.Data[0].Elevation = &Elevation{
			Feet:   0,
			Meters: 0,
		}
	}

	if data.Data[0].Humidity == nil {
		log.Println("No humidity data, defaulting to 0%")
		data.Data[0].Humidity = &Humidity{
			Percent: 0,
		}
	}

	if data.Data[0].Temperature == nil {
		log.Println("No temperature data, defaulting to 15 Celsius")
		data.Data[0].Temperature = &Temperature{
			Celsius:    15,
			Fahrenheit: 59,
		}
	}

	if data.Data[0].Visibility == nil {
		log.Println("No visibility data, defaulting to 10+ SM")
		data.Data[0].Visibility = &Visibility{
			Miles:       "Greater than 10 miles",
			MilesFloat:  10,
			Meters:      "Greater than 9000 meters",
			MetersFloat: 9000,
		}
	}

	if data.Data[0].Wind == nil {
		log.Println("No wind data, defaulting to calm")
		data.Data[0].Wind = &Wind{
			Degrees:  0,
			SpeedKPH: 0,
			SpeedKTS: 0,
			SpeedMPH: 0,
			SpeedMPS: 0,
			GustKPH:  0,
			GustKTS:  0,
			GustMPH:  0,
			GustMPS:  0,
		}
	}

	return nil
}

// GenerateMETAR generates a metar based on the weather settings added to the
// DCS miz
func GenerateMETAR(wx WeatherData, rmk string) (string, error) {
	var data Data
	if len(wx.Data) > 1 {
		data = wx.Data[1]
	} else {
		data = wx.Data[0]
	}

	var metar string

	// add ICAO
	metar += "METAR: " + data.ICAO + " "

	// get observed time, no need to translate time zone since it's in Zulu
	t, err := time.Parse("2006-01-02T15:04:05Z", data.Observed)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05", data.Observed)
		if err != nil {
			return "", fmt.Errorf("Error parsing METAR time: %v", err)
		}
	}
	// want format DDHHMMZ
	metar += fmt.Sprintf("%02d%02d%02dZ ", t.Day(), t.Hour(), t.Minute())

	// winds DIRSPDKT
	if data.Wind.GustKTS > 0 {
		metar += fmt.Sprintf(
			"%03d%02dG%02dKT ",
			int(data.Wind.Degrees),
			int(data.Wind.SpeedKTS),
			int(data.Wind.GustKTS),
		)
	} else {
		metar += fmt.Sprintf(
			"%03d%02dKT ",
			int(data.Wind.Degrees),
			int(data.Wind.SpeedKTS),
		)
	}

	// visibility
	metar += fmt.Sprintf("%dSM ", int(data.Visibility.MilesFloat))

	// conditions
	for _, cond := range data.Conditions {
		metar += fmt.Sprintf("%s ", cond.Code)
	}

	// clouds
	if SelectedPreset == "" {
		metar += "CLR "
	} else if clouds, ok := DecodePreset[SelectedPreset]; ok {
		for i, cld := range clouds {
			if i == 0 {
				// convert base to hundreds of feet
				base := int(float64(SelectedBase)*3.28+50) / 100
				metar += fmt.Sprintf("%s%03d ", cld.Name, base)
			} else {
				metar += fmt.Sprintf("%s%s ", cld.Name, cld.Base)
			}
		}
	} else {
		// using legacy/custom wx
		cloudKind := SelectedPreset[7:10]
		// convert base to hundreds of feet
		base := int(float64(SelectedBase)*3.28+50) / 100
		metar += fmt.Sprintf("%s%03d ", cloudKind, base)
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

	// nosig because usually not updated until 4 hours (whenever rw gets run)
	metar += "NOSIG"

	// rmks
	if rmk != "" {
		metar += " " + rmk
	}

	return metar, nil
}

type WeatherData struct {
	Data       []Data `json:"data,omitempty"`
	NumResults int    `json:"results,omitempty"`
}

type Data struct {
	Barometer      *Barometer   `json:"barometer,omitempty"`
	Ceiling        *Ceiling     `json:"ceiling,omitempty"`
	Clouds         []Clouds     `json:"clouds,omitempty"`
	Conditions     []Conditions `json:"conditions,omitempty"`
	Dewpoint       *Dewpoint    `json:"dewpoint,omitempty"`
	Elevation      *Elevation   `json:"elevation,omitempty"`
	FlightCategory string       `json:"flight_category,omitempty"`
	Humidity       *Humidity    `json:"humidity,omitempty"`
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
	Hg  float64 `json:"hg,omitempty"`
	HPa float64 `json:"hpa,omitempty"`
	KPa float64 `json:"kpa,omitempty"`
	MB  float64 `json:"mb,omitempty"`
}

type Ceiling struct {
	BaseFeetAGL   float64 `json:"base_feet_agl,omitempty"`
	BaseMetersAGL float64 `json:"base_meters_agl,omitempty"`
	Code          string  `json:"code,omitempty"`
	Feet          float64 `json:"feet,omitempty"`
	Meters        float64 `json:"meters,omitempty"`
	Text          string  `json:"text,omitempty"`
}

type Clouds struct {
	BaseFeetAGL   float64 `json:"base_feet_agl,omitempty"`
	BaseMetersAGL float64 `json:"base_meters_agl,omitempty"`
	Code          string  `json:"code,omitempty"`
	Feet          float64 `json:"feet,omitempty"`
	Meters        float64 `json:"meters,omitempty"`
	Text          string  `json:"text,omitempty"`
}

type Conditions struct {
	Code string `json:"code,omitempty"`
	Text string `json:"text,omitempty"`
}

type Dewpoint struct {
	Celsius    float64 `json:"celsius,omitempty"`
	Fahrenheit float64 `json:"fahrenheit,omitempty"`
}

type Elevation struct {
	Feet   float64 `json:"feet,omitempty"`
	Meters float64 `json:"meters,omitempty"`
}

type Humidity struct {
	Percent float64 `json:"percent,omitempty"`
}

type Station struct {
	Location string    `json:"location,omitempty"`
	Name     string    `json:"name,omitempty"`
	Type     string    `json:"type,omitempty"`
	Geometry *Geometry `json:"geometry,omitempty"`
}

type Geometry struct {
	Coordinates []float64 `json:"coordinates,omitempty"`
	Type        string    `json:"type,omitempty"`
}

type Temperature struct {
	Celsius    float64 `json:"celsius,omitempty"`
	Fahrenheit float64 `json:"fahrenheit,omitempty"`
}

type Visibility struct {
	Meters      string  `json:"meters,omitempty"`
	MetersFloat float64 `json:"meters_float,omitempty"`
	Miles       string  `json:"miles,omitempty"`
	MilesFloat  float64 `json:"miles_float,omitempty"`
}

type Wind struct {
	Degrees  float64 `json:"degrees,omitempty"`
	SpeedKPH float64 `json:"speed_kph,omitempty"`
	SpeedKTS float64 `json:"speed_kts,omitempty"`
	SpeedMPH float64 `json:"speed_mph,omitempty"`
	SpeedMPS float64 `json:"speed_mps,omitempty"`
	GustKPH  float64 `json:"gust_kph,omitempty"`
	GustKTS  float64 `json:"gust_kts,omitempty"`
	GustMPH  float64 `json:"gust_mph,omitempty"`
	GustMPS  float64 `json:"gust_mps,omitempty"`
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
	"SCT+RA": {
		{`"RainyPreset4"`, 1260, 4200},  // Light Rain 1
		{`"NEWRAINPRESET4"`, 840, 5174}, // Light Rain 4
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
	"BKN+RA": {
		{`"RainyPreset5"`, 1260, 2520}, // Light Rain 2
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
		{`"RainyPreset1"`, 420, 2940},  // Overcast And Rain 1
		{`"RainyPreset2"`, 840, 2520},  // Overcast And Rain 2
		{`"RainyPreset3"`, 840, 2520},  // Overcast And Rain 3
		{`"RainyPreset6"`, 1260, 2940}, // Light Rain 3
	},
}

type Cloud struct {
	Name string
	Base string
}

var DecodePreset = map[string][]Cloud{
	`"Preset1"`:        {{"FEW", "070"}},
	`"Preset2"`:        {{"FEW", "080"}, {"SCT", "230"}},
	`"Preset3"`:        {{"SCT", "080"}, {"FEW", "210"}},
	`"Preset4"`:        {{"SCT", "080"}, {"SCT", "240"}},
	`"Preset5"`:        {{"SCT", "140"}, {"FEW", "270"}, {"BKN", "400"}},
	`"Preset6"`:        {{"SCT", "080"}, {"FEW", "400"}},
	`"Preset7"`:        {{"BKN", "075"}, {"SCT", "210"}, {"SCT", "400"}},
	`"Preset8"`:        {{"SCT", "180"}, {"FEW", "360"}, {"FEW", "400"}},
	`"Preset9"`:        {{"BKN", "075"}, {"SCT", "200"}, {"FEW", "410"}},
	`"Preset10"`:       {{"SCT", "180"}, {"FEW", "360"}, {"FEW", "400"}},
	`"Preset11"`:       {{"BKN", "180"}, {"BKN", "320"}, {"FEW", "410"}},
	`"Preset12"`:       {{"BKN", "120"}, {"SCT", "220"}, {"FEW", "410"}},
	`"Preset13"`:       {{"BKN", "120"}, {"BKN", "260"}, {"FEW", "410"}},
	`"Preset14"`:       {{"BKN", "070"}, {"FEW", "410"}},
	`"Preset15"`:       {{"SCT", "140"}, {"BKN", "240"}, {"FEW", "400"}},
	`"Preset16"`:       {{"BKN", "140"}, {"BKN", "280"}, {"FEW", "400"}},
	`"Preset17"`:       {{"BKN", "070"}, {"BKN", "200"}, {"BKN", "320"}},
	`"Preset18"`:       {{"BKN", "130"}, {"BKN", "250"}, {"BKN", "380"}},
	`"Preset19"`:       {{"OVC", "090"}, {"BKN", "230"}, {"BKN", "310"}},
	`"Preset20"`:       {{"BKN", "130"}, {"BKN", "280"}, {"FEW", "380"}},
	`"Preset21"`:       {{"BKN", "070"}, {"OVC", "170"}},
	`"Preset22"`:       {{"OVC", "070"}, {"BKN", "170"}},
	`"Preset23"`:       {{"OVC", "110"}, {"BKN", "180"}, {"SCT", "320"}},
	`"Preset24"`:       {{"OVC", "030"}, {"OVC", "170"}, {"BKN", "340"}},
	`"Preset25"`:       {{"OVC", "120"}, {"OVC", "220"}, {"OVC", "400"}},
	`"Preset26"`:       {{"OVC", "090"}, {"BKN", "230"}, {"SCT", "320"}},
	`"Preset27"`:       {{"OVC", "080"}, {"BKN", "250"}, {"BKN", "340"}},
	`"RainyPreset1"`:   {{"OVC", "030"}, {"OVC", "280"}, {"FEW", "400"}},
	`"RainyPreset2"`:   {{"OVC", "030"}, {"SCT", "180"}, {"FEW", "400"}},
	`"RainyPreset3"`:   {{"OVC", "060"}, {"OVC", "190"}, {"SCT", "340"}},
	`"RainyPreset4"`:   {{"SCT", "080"}, {"FEW", "360"}},
	`"RainyPreset5"`:   {{"BKN", "070"}, {"BKN", "200"}, {"BKN", "320"}},
	`"RainyPreset6"`:   {{"OVC", "090"}, {"BKN", "230"}, {"BKN", "310"}},
	`"NEWRAINPRESET4"`: {{"SCT", "080"}, {"SCT", "120"}},
}

var DefaultWeather WeatherData = WeatherData{
	Data: []Data{
		{
			Wind: &Wind{
				SpeedMPS: 1.25,
				Degrees:  270,
				GustMPS:  3,
			},
			Temperature: &Temperature{
				Celsius: 15,
			},
			Barometer: &Barometer{
				Hg: 29.92,
			},
			Clouds: []Clouds{
				{
					Code: "CLR",
				},
			},
			Observed: time.Now().Format("2006-01-02T15:04:05"),
			Station: &Station{
				Geometry: &Geometry{
					Coordinates: []float64{0, 0},
				},
			},
			ICAO: "DGAA",
			Visibility: &Visibility{
				MilesFloat: 10,
			},
			Dewpoint: &Dewpoint{
				Celsius: 10,
			},
		},
	},
	NumResults: 1,
}

type OpenMeteoData struct {
	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
	Elevation        float64 `json:"elevation"`
	GenerationTime   float64 `json:"generationtime_ms"`
	UTCOffsetSeconds int     `json:"utc_offset_seconds"`
	Timezone         string  `json:"timezone"`
	TimezoneAbbr     string  `json:"timezone_abbreviation"`
	Hourly           struct {
		Time              []string  `json:"time"`
		WindSpeed1900     []float64 `json:"windspeed_800hPa"`
		WindSpeed7200     []float64 `json:"windspeed_400hPa"`
		WindDirection1900 []int     `json:"winddirection_800hPa"`
		WindDirection7200 []int     `json:"winddirection_400hPa"`
	} `json:"hourly"`
}

type WindsAloft struct {
	WindSpeed1900     float64
	WindSpeed7200     float64
	WindDirection1900 int
	WindDirection7200 int
}
