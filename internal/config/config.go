package config

import (
	"fmt"
	"os"

	_ "github.com/joho/godotenv/autoload"
)

type Config struct {
	DiscordBotToken string
	GeminiAPIKey    string
	LLMChannelID    string
	GeminiModelName string
}

func Load() (*Config, error) {
	cfg := &Config{
		DiscordBotToken: os.Getenv("DISCORD_BOT_TOKEN"),
		GeminiAPIKey:    os.Getenv("GEMINI_API_STUDIO_KEY"),
		GeminiModelName: os.Getenv("GEMINI_MODEL_NAME"),
	}

	if cfg.DiscordBotToken == "" {
		return nil, fmt.Errorf("DISCORD_BOT_TOKEN not set")
	}
	if cfg.GeminiAPIKey == "" {
		return nil, fmt.Errorf("GEMINI_API_STUDIO_KEY not set")
	}
	
	if cfg.GeminiModelName == "" {
		cfg.GeminiModelName = "gemini-2.5-flash-preview-05-20" // Default if not set
		fmt.Println("GEMINI_MODEL_NAME not set, using default:", cfg.GeminiModelName)
	}

	return cfg, nil
}
