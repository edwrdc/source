package llm

import "context"

type ChatSession interface {
	SendMessage(ctx context.Context, message string) (string, error)
}

type LLMProvider interface {
	GetOrCreateChatSession(ctx context.Context, userID string, systemPrompt string) (ChatSession, error)
	ModelName() string
}
