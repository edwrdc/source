package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/edwrdc/source/internal/bot"
	"github.com/edwrdc/source/internal/config"
	"github.com/edwrdc/source/internal/guildconfig"
	"github.com/edwrdc/source/internal/llm"
)

func Execute() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	guildMgr, err := guildconfig.NewManager("./config_data")
	if err != nil {
		return fmt.Errorf("failed to create guild config manager: %w", err)
	}
	ctx := context.Background()

	geminiProvider, err := llm.NewGeminiProvider(ctx, cfg.GeminiAPIKey, cfg.GeminiModelName)
	if err != nil {
		return fmt.Errorf("failed to create Gemini provider: %w", err)
	}

	discordBot, err := bot.NewBot(cfg, geminiProvider, guildMgr)
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}

	if err := discordBot.Start(); err != nil {
		return fmt.Errorf("failed to start bot: %w", err)
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly shutdown
	if err := discordBot.Stop(); err != nil {
		log.Printf("Error stopping Discord bot: %v", err)
	}
	log.Println("Bot shutdown complete.")
	return nil
}
