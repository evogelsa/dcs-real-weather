package handlers

import (
	"bufio"
	"log"
	"os"
	"regexp"

	dg "github.com/bwmarrin/discordgo"

	"github.com/evogelsa/DCS-real-weather/v2/cmd/bot/config"
)

var reMETAR = regexp.MustCompile(`^\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} METAR: (?P<metar>.*)`)

func LastMETAR(s *dg.Session, i *dg.InteractionCreate) {
	const command = `/last-metar`
	log.Println(command, "called")
	defer timeCommand(command)()

	if ok := verifyCaller(s, i, command, false); !ok {
		return
	}

	cfg := config.Get()

	var server int64
	if len(cfg.Instances) > 1 {
		server = getServer(i)
	}

	rwLogPath := cfg.Instances[server].RealWeatherLog

	f, err := os.Open(rwLogPath)
	if err != nil {
		log.Printf("Unable to open Real Weather log file: %v", err)
		somethingWentWrong(s, i)
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	var metar string
	for sc.Scan() {
		if match := reMETAR.FindStringSubmatch(sc.Text()); len(match) == 2 {
			metar = match[1]
		}
	}

	if metar == "" {
		log.Println("Unable to locate a METAR in your Real Weather log file. Is your configuration correct?")
		s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
			Type: dg.InteractionResponseChannelMessageWithSource,
			Data: &dg.InteractionResponseData{
				Content: "Sorry, a METAR could not be found. Check your log file for more info.",
			},
		})
	}

	s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
		Type: dg.InteractionResponseChannelMessageWithSource,
		Data: &dg.InteractionResponseData{
			Content: metar,
		},
	})
}
