package handlers

import (
	"log"
	"time"

	dg "github.com/bwmarrin/discordgo"
)

func checkForRole(s *dg.Session, i *dg.InteractionCreate, desired string) bool {
	userRoles := i.Interaction.Member.Roles

	guildRoles, err := s.GuildRoles(i.Interaction.GuildID)
	if err != nil {
		log.Printf("Error checking guild roles: %v", err)
		return false
	}

	// get id of desired role
	var id string
	for _, role := range guildRoles {
		if role.Name == desired {
			id = role.ID
			break
		}
	}

	// no match found for desired role
	if id == "" {
		return false
	}

	// check user roles for desired
	for _, role := range userRoles {
		if role == id {
			return true
		}
	}

	// no match found in user roles
	return false
}

func verifyCaller(s *dg.Session, i *dg.InteractionCreate, command string, adminOnly bool) bool {
	// check interaction is of the right type
	if i.Interaction.Type != dg.InteractionApplicationCommand {
		log.Println("unexpected interaction type for " + command)
		return false
	}

	// check command is called in a server
	if i.Interaction.Member == nil {
		// command must be invoked in a guild
		s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
			Type: dg.InteractionResponseChannelMessageWithSource,
			Data: &dg.InteractionResponseData{
				Content: "Sorry, " + command + " cannot be called outside of a guild.",
			},
		})
		return false
	}

	// check user has right permissions for command
	if adminOnly {
		if allowed := checkForRole(s, i, "Real Weather"); !allowed {
			// user is not authorized for command
			s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
				Type: dg.InteractionResponseChannelMessageWithSource,
				Data: &dg.InteractionResponseData{
					Content: "Sorry, you don't have the right permissions for that command (user needs \"Real Weather\" role).",
				},
			})
			return false
		}
	}

	return true
}

func somethingWentWrong(s *dg.Session, i *dg.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
		Type: dg.InteractionResponseChannelMessageWithSource,
		Data: &dg.InteractionResponseData{
			Content: "Sorry, something went wrong. Check your log file and try again.",
		},
	})
}

func timeCommand(command string) func() {
	start := time.Now()
	return func() {
		log.Printf("%s completed in %v", command, time.Since(start))
	}
}

func getServer(i *dg.InteractionCreate) int64 {
	for _, option := range i.ApplicationCommandData().Options {
		if option.Name == "server" {
			return option.IntValue() - 1
		}
	}
	log.Fatalf("Something went terribly wrong, report this as a bug :)")
	return -1
}
