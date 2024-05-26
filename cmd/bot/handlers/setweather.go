package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	dg "github.com/bwmarrin/discordgo"

	"github.com/evogelsa/DCS-real-weather/weather"
)

var isICAO = regexp.MustCompile(`^[A-Z]{4}$`).MatchString

func SetWeather(s *dg.Session, i *dg.InteractionCreate, rwPath string) {
	const command = `/set-weather`
	log.Println(command, "called")
	defer timeCommand(command)()

	if ok := verifyCaller(s, i, command, true); !ok {
		return
	}

	var data weather.WeatherData
	data.Data = make([]weather.Data, 1)

	var response string

	var (
		icao string  // 4 letters only
		baro float64 // inHg
		base float64 // meters
		temp float64 // celsius
		vis  float64 // meters
		wind float64 // meters per second
		gust float64 // meters per second
	)

	for _, option := range i.ApplicationCommandData().Options {
		switch option.Name {
		case "icao":
			icao = option.StringValue()
			icao = strings.ToUpper(icao)

			if isICAO(icao) {
				data.Data[0].ICAO = icao
			} else {
				response = strings.Join([]string{response, "The ICAO you provided was not valid."}, "\n")
			}

		case "baro-hpa":
			baro = option.FloatValue()
			baro *= weather.HPaToInHg
			fallthrough
		case "baro-hg":
			if baro == 0 {
				// not a fallthrough from above
				baro = option.FloatValue()
			}
			data.Data[0].Barometer = &weather.Barometer{Hg: baro}

		case "cloud-code":
			if data.Data[0].Clouds == nil {
				data.Data[0].Clouds = []weather.Clouds{
					{
						Code: option.StringValue(),
					},
				}
			} else {
				data.Data[0].Clouds[0].Code = option.StringValue()
			}

		case "cloud-base-ft":
			base = option.FloatValue()
			base *= weather.FeetToMeters
			fallthrough
		case "cloud-base-m":
			if base == 0 {
				// not a fallthrough
				base = option.FloatValue()
			}
			if data.Data[0].Clouds == nil {
				data.Data[0].Clouds = []weather.Clouds{
					{
						Meters: base,
					},
				}
			} else {
				data.Data[0].Clouds[0].Meters = base
			}

		case "temperature-f":
			temp = option.FloatValue()
			temp = weather.FahrenheitToCelsius(temp)
			fallthrough
		case "temperature-c":
			if temp == 0 {
				temp = option.FloatValue()
			}
			data.Data[0].Temperature = &weather.Temperature{Celsius: temp}

		case "visibility-ft":
			vis = option.FloatValue()
			vis *= weather.FeetToMeters
			fallthrough
		case "visiblity-m":
			if vis == 0 {
				// not a fallthrough
				vis = option.FloatValue()
			}
			data.Data[0].Visibility = &weather.Visibility{MetersFloat: vis}

		case "wind-direction":
			if data.Data[0].Wind == nil {
				data.Data[0].Wind = &weather.Wind{Degrees: option.FloatValue()}
			} else {
				data.Data[0].Wind.Degrees = option.FloatValue()
			}

		case "wind-kt":
			wind = option.FloatValue()
			wind *= weather.KtToMPS
			fallthrough
		case "wind-mps":
			if wind == 0 {
				// not a fallthrough
				wind = option.FloatValue()
			}

			if data.Data[0].Wind == nil {
				data.Data[0].Wind = &weather.Wind{SpeedMPS: wind}
			} else {
				data.Data[0].Wind.SpeedMPS = wind
			}

		case "wind-gust-kt":
			gust = option.FloatValue()
			gust *= weather.KtToMPS
			fallthrough
		case "wind-gust-mps":
			if gust == 0 {
				// not a fallthrough
				gust = option.FloatValue()
			}

			if data.Data[0].Wind == nil {
				data.Data[0].Wind = &weather.Wind{GustMPS: gust}
			} else {
				data.Data[0].Wind.GustMPS = gust
			}

		case "conditions":
			fallthrough
		case "precipitation":
			data.Data[0].Conditions = append(
				data.Data[0].Conditions,
				weather.Conditions{
					Code: option.StringValue(),
				},
			)
		}
	}

	b, err := json.MarshalIndent(&data, "", "  ")
	if err != nil {
		log.Printf("error marshalling weather data: %v", err)
		response = strings.Join([]string{response, "Unknown error, check bot logs."}, "\n")
	} else {
		// write .rwbot
		path := filepath.Join(rwPath, ".rwbot")
		if err := os.WriteFile(path, []byte{}, os.ModePerm); err != nil {
			log.Printf("error .rwbot: %v", err)
			response = strings.Join([]string{response, "Unknown error, check bot logs."}, "\n")
		}

		// write checkwx.json
		path = filepath.Join(rwPath, "checkwx.json")
		if err := os.WriteFile(path, b, os.ModePerm); err != nil {
			log.Printf("error writing weather data: %v", err)
			response = strings.Join([]string{response, "Unknown error, check bot logs."}, "\n")
		} else {
			log.Println("/set-weather wrote custom weather data")
		}
	}

	if response == "" {
		log.Println("/set-weather generated weather data with no errors")
		response = "Your custom weather was successfully generated. It will be used next the time Real Weather is run."
	} else {
		log.Println("/set-weather generated weather data with errors")
		response = fmt.Sprintf(
			"There were some errors generating your weather:"+
				"\n"+
				"%s"+
				"\n\n"+
				"Custom weather has still been generated if there were valid parameters."+
				" It will be used the next time Real Weather is run."+
				" You can regenerate the weather to fix any errors by using this command again.",
			response,
		)
	}

	s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
		Type: dg.InteractionResponseChannelMessageWithSource,
		Data: &dg.InteractionResponseData{
			Content: response,
		},
	})
}
