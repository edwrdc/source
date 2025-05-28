package llm

import (
	"context"
	"fmt"
	"log"
	"sync"

	"google.golang.org/genai"
)

type geminiChatSession struct {
	genaiSession *genai.Chat
}

func (gcs *geminiChatSession) SendMessage(ctx context.Context, message string) (string, error) {
	if gcs.genaiSession == nil {
		return "", fmt.Errorf("Gemini session is not initialized")
	}

	resp, err := gcs.genaiSession.SendMessage(ctx, genai.Part{Text: message})
	if err != nil {
		return "", fmt.Errorf("Gemini API SendMessage error: %w", err)
	}

	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("Gemini API returned no valid candidates")
	}

	// Get the text from the first part
	if part := resp.Candidates[0].Content.Parts[0]; part != nil && part.Text != "" {
		return part.Text, nil
	}
	return "", fmt.Errorf("Gemini API response part was not text")
}

type GeminiProvider struct {
	client        *genai.Client
	modelName     string
	conversations map[string]*geminiChatSession
	mu            sync.RWMutex
}

func NewGeminiProvider(ctx context.Context, apiKey string, modelName string) (*GeminiProvider, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating Gemini client: %w", err)
	}

	return &GeminiProvider{
		client:        client,
		modelName:     modelName,
		conversations: make(map[string]*geminiChatSession),
	}, nil
}

func (gp *GeminiProvider) GetOrCreateChatSession(ctx context.Context, userID string, systemPrompt string) (ChatSession, error) {
	gp.mu.RLock()
	session, found := gp.conversations[userID]
	gp.mu.RUnlock()

	if found {
		return session, nil
	}

	gp.mu.Lock()
	defer gp.mu.Unlock()

	if s, ok := gp.conversations[userID]; ok {
		return s, nil
	}

	log.Printf("Creating new Gemini chat session for user %s with model %s", userID, gp.modelName)

	var config *genai.GenerateContentConfig
	if systemPrompt != "" {
		config = &genai.GenerateContentConfig{
			SystemInstruction: &genai.Content{
				Parts: []*genai.Part{{Text: systemPrompt}},
			},
		}
	}

	genaiChat, err := gp.client.Chats.Create(ctx, gp.modelName, config, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating chat session: %w", err)
	}

	newSession := &geminiChatSession{genaiSession: genaiChat}
	gp.conversations[userID] = newSession
	return newSession, nil
}

func (gp *GeminiProvider) ModelName() string {
	return gp.modelName
}
