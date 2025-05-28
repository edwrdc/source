package bot

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) readyHandler(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Bot %s is ready.", event.User.Username)

	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "set-llm-channel",
			Description: "Sets the channel for LLM interactions in this server.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionChannel,
					Name:         "channel",
					Description:  "The text channel to use for the LLM bot.",
					ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText}, // Only text channels
					Required:     true,
				},
			},
		},
		{
			Name:        "remove-llm-channel",
			Description: "Removes the configured LLM interaction channel for this server.",
		},
	}

	registeredCommands, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, "", commands)
	if err != nil {
		log.Fatalf("Cannot register slash commands: %v", err)
	}
	b.registeredCommands = registeredCommands
	log.Printf("Registered %d slash commands.", len(registeredCommands))
}

// slash command handler
func (b *Bot) interactionCreateHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommand {
		// need "Manage Channel" or "Administrator" permissions.
		perms, err := s.UserChannelPermissions(i.Member.User.ID, i.ChannelID)
		if err != nil {
			log.Printf("Error getting user permissions for %s in channel %s: %v", i.Member.User.ID, i.ChannelID, err)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "Error: Could not verify your permissions."},
			})
			return
		}

		if i.GuildID == "" {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "This command can only be used in a server."},
			})
			return
		}

		isAdmin := (perms&discordgo.PermissionAdministrator == discordgo.PermissionAdministrator)
		canManageChannels := (perms&discordgo.PermissionManageChannels == discordgo.PermissionManageChannels)

		switch i.ApplicationCommandData().Name {
		case "set-llm-channel":
			if !isAdmin && !canManageChannels {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You need 'Manage Channels' or 'Administrator' permission to use this command.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}
			b.handleSetLLMChannel(s, i)
		case "remove-llm-channel":
			if !isAdmin && !canManageChannels {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You need 'Manage Channels' or 'Administrator' permission to use this command.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}
			b.handleRemoveLLMChannel(s, i)
		}
	}
}

func (b *Bot) handleSetLLMChannel(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	var channelID string
	for _, opt := range options {
		if opt.Name == "channel" {
			channelID = opt.ChannelValue(s).ID
			break
		}
	}

	if channelID == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Error: Channel option not found."},
		})
		return
	}

	err := b.guildConfigMgr.SetLLMChannel(i.GuildID, channelID)
	responseContent := ""
	if err != nil {
		responseContent = fmt.Sprintf("Error setting LLM channel: %v", err)
		log.Printf("Error in SetLLMChannel for guild %s: %v", i.GuildID, err)
	} else {
		responseContent = fmt.Sprintf("LLM interaction channel set to <#%s> for this server.", channelID)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: responseContent},
	})
}

func (b *Bot) handleRemoveLLMChannel(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := b.guildConfigMgr.RemoveLLMChannel(i.GuildID)
	responseContent := ""
	if err != nil {
		responseContent = fmt.Sprintf("Error removing LLM channel configuration: %v", err)
		log.Printf("Error in RemoveLLMChannel for guild %s: %v", i.GuildID, err)
	} else {
		responseContent = "LLM interaction channel configuration has been removed for this server. The bot will no longer respond to messages unless reconfigured."
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: responseContent},
	})
}
