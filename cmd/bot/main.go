package main

import (
	"io"
	"log"
	"os"
	"os/signal"

	dg "github.com/bwmarrin/discordgo"

	"github.com/evogelsa/DCS-real-weather/cmd/bot/config"
	"github.com/evogelsa/DCS-real-weather/cmd/bot/handlers"
)

var (
	cfg      config.Configuration
	guildID  string
	botToken string
)

// init initializes the configuration
func init() {
	cfg = config.Get()

	// set log file if configured
	if cfg.Log != "" {
		f, err := os.OpenFile(
			cfg.Log,
			os.O_WRONLY|os.O_CREATE|os.O_APPEND,
			0644,
		)
		if err != nil {
			log.Printf("Error opening log file: %v\n", err)
		}
		// defer f.Close() let file be closed when program exits

		mw := io.MultiWriter(os.Stdout, f)

		log.SetOutput(mw)
	}
}

var s *dg.Session

// init initializes the bot session
func init() {
	var err error
	s, err = dg.New("Bot " + cfg.BotToken)
	if err != nil {
		log.Fatalf("Error registering new bot: %v", err)
	}
}

var (
	commands = []*dg.ApplicationCommand{
		{
			Name:        "set-weather",
			Description: "set custom weather for the next time Real Weather runs",
			Options: []*dg.ApplicationCommandOption{
				{
					Type:        dg.ApplicationCommandOptionString,
					Name:        "icao",
					Description: "airport ICAO to use",
					Required:    false,
				},
				{
					Type:        dg.ApplicationCommandOptionNumber,
					Name:        "baro-hg",
					Description: "barometer setting in inHg",
					Required:    false,
				},
				{
					Type:        dg.ApplicationCommandOptionNumber,
					Name:        "baro-hpa",
					Description: "barometer setting in hPa",
					Required:    false,
				},
				{
					Type:        dg.ApplicationCommandOptionString,
					Name:        "cloud-code",
					Description: "cloud code for first cloud layer",
					Required:    false,
					Choices: []*dg.ApplicationCommandOptionChoice{
						{
							Name:  "Overcast",
							Value: "OVC",
						},
						{
							Name:  "Broken",
							Value: "BKN",
						},
						{
							Name:  "Scattered",
							Value: "SCT",
						},
						{
							Name:  "Few",
							Value: "FEW",
						},
						{
							Name:  "Clear",
							Value: "CLR",
						},
					},
				},
				{
					Type:        dg.ApplicationCommandOptionNumber,
					Name:        "cloud-base-m",
					Description: "cloud base in meters",
					Required:    false,
				},
				{
					Type:        dg.ApplicationCommandOptionNumber,
					Name:        "cloud-base-ft",
					Description: "cloud base in feet",
					Required:    false,
				},
				{
					Type:        dg.ApplicationCommandOptionNumber,
					Name:        "temperature-c",
					Description: "temperature in Celsius",
					Required:    false,
				},
				{
					Type:        dg.ApplicationCommandOptionNumber,
					Name:        "temperature-f",
					Description: "temperature in Fahrenheit",
					Required:    false,
				},
				{
					Type:        dg.ApplicationCommandOptionNumber,
					Name:        "visibility-m",
					Description: "visibility in meters",
					Required:    false,
				},
				{
					Type:        dg.ApplicationCommandOptionNumber,
					Name:        "visibility-ft",
					Description: "visibility in feet",
					Required:    false,
				},
				{
					Type:        dg.ApplicationCommandOptionNumber,
					Name:        "wind-direction",
					Description: "azimuth direction the wind is from",
					Required:    false,
				},
				{
					Type:        dg.ApplicationCommandOptionNumber,
					Name:        "wind-kt",
					Description: "wind speed in knots",
					Required:    false,
				},
				{
					Type:        dg.ApplicationCommandOptionNumber,
					Name:        "wind-mps",
					Description: "wind speed in meters per second",
					Required:    false,
				},
				{
					Type:        dg.ApplicationCommandOptionNumber,
					Name:        "wind-gust-kt",
					Description: "wind gust speed in knots",
					Required:    false,
				},
				{
					Type:        dg.ApplicationCommandOptionNumber,
					Name:        "wind-gust-mps",
					Description: "wind gust speed in meters per second",
					Required:    false,
				},
				{
					Type:        dg.ApplicationCommandOptionString,
					Name:        "conditions",
					Description: "condition code",
					Required:    false,
					Choices: []*dg.ApplicationCommandOptionChoice{
						{
							Name:  "Fog",
							Value: "FG",
						},
						{
							Name:  "Dust",
							Value: "DS",
						},
						{
							Name:  "None",
							Value: "",
						},
					},
				},
				{
					Type:        dg.ApplicationCommandOptionString,
					Name:        "precipitation",
					Description: "precipitation code",
					Required:    false,
					Choices: []*dg.ApplicationCommandOptionChoice{
						{
							Name:  "Rain",
							Value: "RA",
						},
						{
							Name:  "Thunderstorm",
							Value: "TS",
						},
					},
				},
			},
		},
		{
			Name:        "last-metar",
			Description: "fetches the last METAR generated by Real Weather",
		},
	}

	commandHandlers = map[string]func(s *dg.Session, i *dg.InteractionCreate){
		"set-weather": handlers.SetWeather,
		"last-metar":  handlers.LastMETAR,
	}
)

// adjust command signatures if using multi instances
func init() {
	if len(cfg.Instances) > 1 {
		var choices []*dg.ApplicationCommandOptionChoice
		for i, instance := range cfg.Instances {
			choices = append(choices, &dg.ApplicationCommandOptionChoice{Name: instance.Name, Value: i})
		}
		for i := range commands {
			commands[i].Options = append(
				[]*dg.ApplicationCommandOption{
					{
						Type:        dg.ApplicationCommandOptionInteger,
						Name:        "server",
						Description: "which server instance to use (first instance is 1)",
						Required:    true,
						Choices:     choices,
					},
				},
				commands[i].Options...,
			)
		}
	}
}

// add command handlers
func init() {
	s.AddHandler(func(s *dg.Session, i *dg.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}

func main() {
	s.AddHandler(func(s *dg.Session, r *dg.Ready) {
		log.Printf(
			"Logged in as: %v#%v",
			s.State.User.Username,
			s.State.User.Discriminator,
		)
	})
	err := s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}
	defer s.Close()

	log.Println("Unregistering old commands...")
	registeredCommands, err := s.ApplicationCommands(s.State.User.ID, guildID)
	if err != nil {
		log.Printf("Unable to unregister commands: %v", err)
	} else {
		for _, v := range registeredCommands {
			err := s.ApplicationCommandDelete(s.State.User.ID, guildID, v.ID)
			if err != nil {
				log.Printf("Unable to unregister command: %s", v.Name)
			} else {
				log.Printf("Command unregistered: %s", v.Name)
			}
		}
	}

	log.Println("Registering commands...")
	for _, v := range commands {
		_, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, v)
		if err != nil {
			log.Printf("Failed to register command %s: %v", v.Name, err)
		} else {
			log.Printf("Registered command: %s", v.Name)
		}
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Bot initialization completed.")
	log.Println("Press Ctrl+C to exit")
	<-stop
	log.Println("Shutting down...")
}
