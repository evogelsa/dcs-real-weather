package weather

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"
)

type altWeather struct {
	Temp       *float64     `json:"temp,omitempty"`
	Dew        *float64     `json:"dewp,omitempty"`
	WindDir    *float64     `json:"wdir,omitempty"`
	WindSpeed  *float64     `json:"wspd,omitempty"`
	WindGust   *float64     `json:"wgst,omitempty"`
	Visibility *json.Number `json:"visib,string,omitempty"`
	Altimeter  *float64     `json:"altim,omitempty"`
	Conditions *string      `json:"wxString,omitempty"`
	Clouds     []altClouds  `json:"clouds,omitempty"`
	Latitude   *float64     `json:"lat,omitempty"`
	Longitude  *float64     `json:"lon,omitempty"`
	ReportTime *string      `json:"reportTime,omitempty"`
}

type altClouds struct {
	Cover *string  `json:"cover,omitempty"`
	Base  *float64 `json:"base,omitempty"`
}

func getWeatherAlternate(icao string) (WeatherData, error) {
	log.Println("Getting weather from Aviation Weather...")

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
			"Error making request to Aviation Weather: %v",
			err,
		)
	}

	// verify response OK
	if resp.StatusCode != http.StatusOK {
		return WeatherData{}, fmt.Errorf("Aviation Weather bad status: %v", resp.Status)
	}
	defer resp.Body.Close()

	// parse response byte array
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return WeatherData{}, fmt.Errorf(
			"Error parsing Aviation Weather response: %v",
			err,
		)
	}

	log.Println("Got weather data:", string(body))
	log.Println("Parsing weather...")

	// format json response into weatherdata struct
	var intermediate []altWeather
	err = json.Unmarshal(body, &intermediate)
	if err != nil {
		return WeatherData{}, err
	}

	if len(intermediate) < 1 {
		return WeatherData{}, fmt.Errorf(
			"Aviation Weather returned no results for ICAO %s",
			icao,
		)
	}

	// convert to WeatherData format
	var res WeatherData

	res.NumResults = 1
	res.Data = make([]Data, 1)

	if intermediate[0].Temp != nil {
		res.Data[0].Temperature = &Temperature{Celsius: *intermediate[0].Temp}
	}

	if intermediate[0].Dew != nil {
		res.Data[0].Dewpoint = &Dewpoint{Celsius: *intermediate[0].Dew}
	}

	res.Data[0].Wind = &Wind{}

	if intermediate[0].WindDir != nil {
		res.Data[0].Wind.Degrees = *intermediate[0].WindDir
	}

	if intermediate[0].WindSpeed != nil {
		res.Data[0].Wind.SpeedKTS = *intermediate[0].WindSpeed
	}

	if intermediate[0].WindGust != nil {
		res.Data[0].Wind.GustKTS = *intermediate[0].WindGust
	}

	if intermediate[0].Visibility != nil {
		res.Data[0].Visibility = &Visibility{}

		if v, err := intermediate[0].Visibility.Float64(); err == nil {
			res.Data[0].Visibility.MetersFloat = v * MilesToMeters
		} else {
			var vis int
			n, err := fmt.Sscanf(intermediate[0].Visibility.String(), "%d+", &vis)
			if n == 1 && err == nil {
				res.Data[0].Visibility.MetersFloat = float64(vis) * MilesToMeters
			} else {
				log.Printf("Failed to parse visibility from Aviation Weather: %v", err)
				res.Data[0].Visibility.MetersFloat = 9000
			}
		}
	}

	if intermediate[0].Altimeter != nil {
		res.Data[0].Barometer = &Barometer{Hg: *intermediate[0].Altimeter * HPaToInHg}
	}

	// checking for conditions this way may add some repeat/redundant codes, but
	// the way the codes are used means it doesn't actually matter as long as
	// at least one code of each condition is added
	if intermediate[0].Conditions != nil {
		// check for fog
		for _, code := range FogCodes() {
			if strings.Contains(*intermediate[0].Conditions, code) {
				res.Data[0].Conditions = append(res.Data[0].Conditions, Conditions{Code: code})
			}
		}

		// check for dust
		for _, code := range DustCodes() {
			if strings.Contains(*intermediate[0].Conditions, code) {
				res.Data[0].Conditions = append(res.Data[0].Conditions, Conditions{Code: code})
			}
		}

		// check for storms
		for _, code := range StormCodes() {
			if strings.Contains(*intermediate[0].Conditions, code) {
				res.Data[0].Conditions = append(res.Data[0].Conditions, Conditions{Code: code + "RA"})
			}
		}

		// check for non storm precip
		for _, code := range PrecipCodes() {
			if strings.Contains(*intermediate[0].Conditions, code) {
				res.Data[0].Conditions = append(res.Data[0].Conditions, Conditions{Code: code})
			}
		}
	}

	if len(intermediate[0].Clouds) > 0 {
		for _, cloud := range intermediate[0].Clouds {
			if cloud.Cover == nil && cloud.Base == nil {
				// unparsable cloud data
				continue
			} else if cloud.Cover != nil && slices.Contains(ClearCodes(), *cloud.Cover) {
				// clear cloud
				res.Data[0].Clouds = append(
					res.Data[0].Clouds,
					Clouds{Code: *cloud.Cover, Meters: 0},
				)
			} else if cloud.Base != nil {
				res.Data[0].Clouds = append(
					res.Data[0].Clouds,
					Clouds{Code: *cloud.Cover, Meters: *cloud.Base * FeetToMeters},
				)
			}
		}
	}

	if intermediate[0].Latitude != nil && intermediate[0].Longitude != nil {
		res.Data[0].Station = &Station{
			Geometry: &Geometry{
				Coordinates: []float64{*intermediate[0].Longitude, *intermediate[0].Latitude},
			},
		}
	}

	if intermediate[0].ReportTime != nil {
		t, err := time.Parse("2006-01-02 15:04:05", *intermediate[0].ReportTime)
		if err == nil {
			res.Data[0].Observed = t.Format("2006-01-02T15:04:05")
		}
	}

	res.Data[0].ICAO = strings.ToUpper(icao)

	if err := ValidateWeather(&res); err != nil {
		return res, err
	}

	log.Println("Parsed weather")

	return res, nil
}
