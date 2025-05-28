package bot

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/edwrdc/source/internal/config"
	"github.com/edwrdc/source/internal/guildconfig"
	"github.com/edwrdc/source/internal/llm"
)

type Bot struct {
	discord            *discordgo.Session
	llmProvider        llm.LLMProvider
	cfg                *config.Config
	guildConfigMgr     *guildconfig.Manager
	registeredCommands []*discordgo.ApplicationCommand
}

func NewBot(cfg *config.Config, provider llm.LLMProvider, guildMgr *guildconfig.Manager) (*Bot, error) {
	dg, err := discordgo.New("Bot " + cfg.DiscordBotToken)
	if err != nil {
		return nil, fmt.Errorf("error creating Discord session: %w", err)
	}

	botInstance := &Bot{
		discord:        dg,
		llmProvider:    provider,
		cfg:            cfg,
		guildConfigMgr: guildMgr,
	}

	dg.AddHandler(botInstance.readyHandler)
	dg.AddHandler(botInstance.llmMessageCreateHandler)
	dg.AddHandler(botInstance.interactionCreateHandler)

	dg.Identify.Intents = discordgo.IntentGuilds | discordgo.IntentGuildMessages | discordgo.IntentMessageContent
	return botInstance, nil
}

func (b *Bot) Start() error {
	log.Println("Bot is starting...")
	err := b.discord.Open()
	if err != nil {
		return fmt.Errorf("error opening connection: %w", err)
	}
	log.Println("LLM Bot is running. Press CTRL-C to exit.")
	return nil
}

func (b *Bot) Stop() error {
	log.Println("Bot is shutting down...")
	if b.discord != nil {
		return b.discord.Close()
	}
	return nil
}
