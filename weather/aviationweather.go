package weather

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/goccy/go-yaml"

	"github.com/evogelsa/DCS-real-weather/v2/logger"
)

type aviationWeatherData struct {
	Temp       *float64                `yaml:"temp,omitempty"`
	Dew        *float64                `yaml:"dewp,omitempty"`
	WindDir    *string                 `yaml:"wdir,omitempty"`
	WindSpeed  *float64                `yaml:"wspd,omitempty"`
	WindGust   *float64                `yaml:"wgst,omitempty"`
	Visibility *string                 `yaml:"visib,omitempty"`
	Altimeter  *float64                `yaml:"altim,omitempty"`
	Conditions *string                 `yaml:"wxString,omitempty"`
	Clouds     []aviationWeatherClouds `yaml:"clouds,omitempty"`
	Latitude   *float64                `yaml:"lat,omitempty"`
	Longitude  *float64                `yaml:"lon,omitempty"`
	ReportTime *string                 `yaml:"reportTime,omitempty"`
}

type aviationWeatherClouds struct {
	Cover *string  `json:"cover,omitempty"`
	Base  *float64 `json:"base,omitempty"`
}

func getWeatherAviationWeather(icao string) (WeatherData, error) {
	logger.Infoln("getting weather from aviation weather...")

	// create http client to fetch weather data, timeout after 5 sec
	timeout := time.Duration(5 * time.Second)
	client := http.Client{Timeout: timeout}

	request, err := http.NewRequest(
		"GET",
		"https://aviationweather.gov/cgi-bin/data/metar.php?ids="+icao+"&format=json",
		nil,
	)
	if err != nil {
		return WeatherData{}, err
	}

	// make api request
	resp, err := client.Do(request)
	if err != nil {
		return WeatherData{}, fmt.Errorf(
			"error making request to Aviation Weather: %v",
			err,
		)
	}

	// verify response OK
	if resp.StatusCode != http.StatusOK {
		return WeatherData{}, fmt.Errorf("aviation weather bad status: %v", resp.Status)
	}
	defer resp.Body.Close()

	// parse response byte array
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return WeatherData{}, fmt.Errorf(
			"error parsing aviation weather response: %v",
			err,
		)
	}

	logger.Infoln("got weather data:", string(body))
	logger.Infoln("parsing weather...")

	// format json response into weatherdata struct
	var intermediate []aviationWeatherData
	err = yaml.Unmarshal(body, &intermediate)
	if err != nil {
		return WeatherData{}, err
	}

	if len(intermediate) < 1 {
		return WeatherData{}, fmt.Errorf(
			"aviation weather returned no results for icao \"%s\"",
			icao,
		)
	}

	res := convertAviationWeather(intermediate)

	res.Data[0].ICAO = strings.ToUpper(icao)

	return res, nil
}

// convertAviationWeather converts aviationWeatherData to WeatherData
func convertAviationWeather(data []aviationWeatherData) WeatherData {
	// convert to WeatherData format
	var res WeatherData

	res.NumResults = 1
	res.Data = make([]Data, 1)

	convertTemperature(&res, data)

	convertDewpoint(&res, data)

	convertWind(&res, data)

	convertVisibility(&res, data)

	convertAltimeter(&res, data)

	convertConditions(&res, data)

	convertClouds(&res, data)

	convertCoordinates(&res, data)

	convertTime(&res, data)

	logger.Infoln("parsed weather")

	return res
}

// convertTemperature converts the temperature to WeatherData
func convertTemperature(out *WeatherData, data []aviationWeatherData) {
	if data[0].Temp != nil {
		out.Data[0].Temperature = &Temperature{Celsius: *data[0].Temp}
	}
}

// convertDewpoint converts the dewpoint
func convertDewpoint(out *WeatherData, data []aviationWeatherData) {
	if data[0].Dew != nil {
		out.Data[0].Dewpoint = &Dewpoint{Celsius: *data[0].Dew}
	}
}

// convertWind converts the wind
func convertWind(out *WeatherData, data []aviationWeatherData) {
	out.Data[0].Wind = &Wind{}

	if data[0].WindDir != nil {
		// winddir may be a number or text for variable, e.g. "VRB"
		// if text, randomize direction, otherwise parse as float
		var v float64
		if err := json.Unmarshal([]byte(*data[0].WindDir), &v); err == nil {
			out.Data[0].Wind.Degrees = v
		} else {
			logger.Infoln("converting variable winds to random direction")
			out.Data[0].Wind.Degrees = float64(rand.Intn(36) * 10)
		}
	}

	if data[0].WindSpeed != nil {
		out.Data[0].Wind.SpeedMPS = *data[0].WindSpeed * KtToMPS
	}

	if data[0].WindGust != nil {
		out.Data[0].Wind.GustMPS = *data[0].WindGust * KtToMPS
	}
}

// convertVisibility converts the visiblity
func convertVisibility(out *WeatherData, data []aviationWeatherData) {
	if data[0].Visibility != nil {
		out.Data[0].Visibility = &Visibility{}

		var vis int
		n, err := fmt.Sscanf(*data[0].Visibility, "%d", &vis)
		if n == 1 && err == nil {
			out.Data[0].Visibility.MetersFloat = float64(vis) * MilesToMeters
		} else {
			n, err := fmt.Sscanf(*data[0].Visibility, "%d+", &vis)
			if n == 1 && err == nil {
				out.Data[0].Visibility.MetersFloat = float64(vis) * MilesToMeters
			} else {
				logger.Errorf("failed to parse visibility from aviation weather: %v", err)
				out.Data[0].Visibility.MetersFloat = 9000
				logger.Warnln("defaulting visibility to 9000 meters")
			}
		}
	}
}

// convertAltimeter converts the altimeter setting
func convertAltimeter(out *WeatherData, data []aviationWeatherData) {
	if data[0].Altimeter != nil {
		out.Data[0].Barometer = &Barometer{Hg: *data[0].Altimeter * HPaToInHg}
	}
}

// convertConditions converts the condition codes
func convertConditions(out *WeatherData, data []aviationWeatherData) {
	// checking for conditions this way may add some repeat/redundant codes, but
	// the way the codes are used means it doesn't actually matter as long as
	// at least one code of each condition is added
	if data[0].Conditions != nil {
		// check for fog
		for _, code := range FogCodes() {
			if strings.Contains(*data[0].Conditions, code) {
				out.Data[0].Conditions = append(out.Data[0].Conditions, Conditions{Code: code})
			}
		}

		// check for dust
		for _, code := range DustCodes() {
			if strings.Contains(*data[0].Conditions, code) {
				out.Data[0].Conditions = append(out.Data[0].Conditions, Conditions{Code: code})
			}
		}

		// check for storms
		for _, code := range StormCodes() {
			if strings.Contains(*data[0].Conditions, code) {
				out.Data[0].Conditions = append(out.Data[0].Conditions, Conditions{Code: code + "RA"})
			}
		}

		// check for non storm precip
		for _, code := range PrecipCodes() {
			if strings.Contains(*data[0].Conditions, code) {
				out.Data[0].Conditions = append(out.Data[0].Conditions, Conditions{Code: code})
			}
		}
	}
}

// convertClouds converts the clouds
func convertClouds(out *WeatherData, data []aviationWeatherData) {
	if len(data[0].Clouds) > 0 {
		for _, cloud := range data[0].Clouds {
			if cloud.Cover == nil && cloud.Base == nil {
				// unparsable cloud data
				continue
			} else if cloud.Cover != nil && slices.Contains(ClearCodes(), *cloud.Cover) {
				// clear cloud
				out.Data[0].Clouds = append(
					out.Data[0].Clouds,
					Clouds{Code: *cloud.Cover, Meters: 0},
				)
			} else if cloud.Cover != nil && cloud.Base != nil {
				// some other cloud, add to cloud list
				out.Data[0].Clouds = append(
					out.Data[0].Clouds,
					Clouds{Code: *cloud.Cover, Meters: *cloud.Base * FeetToMeters},
				)
			}
		}
	}
}

// convertCoordinates converts the lat lon
func convertCoordinates(out *WeatherData, data []aviationWeatherData) {
	if data[0].Latitude != nil && data[0].Longitude != nil {
		out.Data[0].Station = &Station{
			Geometry: &Geometry{
				Coordinates: []float64{*data[0].Longitude, *data[0].Latitude},
			},
		}
	}
}

// convertTime converts the report time
func convertTime(out *WeatherData, data []aviationWeatherData) {
	if data[0].ReportTime != nil {
		t, err := time.Parse("2006-01-02 15:04:05", *data[0].ReportTime)
		if err == nil {
			out.Data[0].Observed = t.Format("2006-01-02T15:04:05")
		}
	}
}
