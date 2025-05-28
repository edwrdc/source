package bot

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/edwrdc/source/internal/llm"
)

const discordMessageLimit = 4000 // discord hard limit

func (b *Bot) llmMessageCreateHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.GuildID == "" {
		return
	}

	configuredLLMChannelID, found := b.guildConfigMgr.GetLLMChannel(m.GuildID)
	if !found {
		return
	}

	if m.ChannelID != configuredLLMChannelID {
		return
	}

	userMessage := strings.TrimSpace(m.Content)
	if userMessage == "" {
		return
	}

	userID := m.Author.ID

	systemInstruction := fmt.Sprintf(
		"You are a helpful assistant on a Discord server. Your name is %s. "+
			"Please keep your answers very concise and to the point, specifically for Discord messages. "+
			"Aim for responses well under %d characters. "+
			"If a topic is complex, provide a summary or the most critical information. "+
			"Avoid lengthy explanations unless explicitly asked to elaborate and even then, be mindful of message limits.",
		s.State.User.Username,
		discordMessageLimit-200, // some buffer
	)

	ctx := context.Background()
	chatSession, err := b.llmProvider.GetOrCreateChatSession(ctx, userID, systemInstruction)
	if err != nil {
		log.Printf("Error getting/creating chat session for user %s: %v", userID, err)
		s.ChannelMessageSendReply(m.ChannelID, "😭 Sorry, I couldn't initialize a chat session for you right now.", &discordgo.MessageReference{
			MessageID: m.ID, ChannelID: m.ChannelID, GuildID: m.GuildID,
		})
		return
	}

	if err = s.ChannelTyping(m.ChannelID); err != nil {
		log.Printf("Error starting typing indicator: %v", err)
	}

	thinkingMsg, err := s.ChannelMessageSendReply(m.ChannelID, "🤔 Bot is thinking...", &discordgo.MessageReference{
		MessageID: m.ID, ChannelID: m.ChannelID, GuildID: m.GuildID,
	})
	if err != nil {
		log.Printf("Error sending thinking message: %v", err)
		s.ChannelMessageSend(m.ChannelID, "😭 Sorry, something went wrong before I could think!")
		return
	}

	go func(cs llm.ChatSession, uMsg, uID, originalMsgID, guildID, thinkingMsgID, channelID string) {
		llmCtx := context.Background()
		llmResponse, err := cs.SendMessage(llmCtx, uMsg)

		if err != nil {
			log.Printf("Error from LLMProvider for user %s: %v", uID, err)
			llmResponse = "😭 Sorry, I encountered an error while processing your request. Please try again."
		} else if llmResponse == "" {
			llmResponse = "😭 I'm sorry, I couldn't generate a response. Please try rephrasing."
		}

		// just incase if the llm doesn't not adhere to the discord length constaraint
		if len(llmResponse) > discordMessageLimit {
			log.Printf("LLM response for user %s exceeded target length (%d chars). Full response: %s", uID, len(llmResponse), llmResponse)

			suffix := "... (response truncated)"
			if len(llmResponse) > discordMessageLimit-len(suffix) {
				llmResponse = llmResponse[:discordMessageLimit-len(suffix)] + suffix
			} else {
				llmResponse = llmResponse[:discordMessageLimit-3] + "..."
			}
		}

		_, editErr := s.ChannelMessageEdit(channelID, thinkingMsgID, llmResponse)
		if editErr != nil {
			log.Printf("Error editing message (ThinkingMsgID: %s, User: %s): %v. LLM Response was: %s", thinkingMsgID, uID, editErr, llmResponse)

			_, sendErr := s.ChannelMessageSendReply(channelID, llmResponse, &discordgo.MessageReference{
				MessageID: originalMsgID, ChannelID: channelID, GuildID: guildID,
			})
			if sendErr != nil {
				log.Printf("Fallback sendReply also failed for user %s: %v", uID, sendErr)
				s.ChannelMessageSendReply(channelID, "😭 I tried to respond. The content might be too long or invalid.", &discordgo.MessageReference{
					MessageID: originalMsgID, ChannelID: channelID, GuildID: guildID,
				})
			}
		}
	}(chatSession, userMessage, userID, m.ID, m.GuildID, thinkingMsg.ID, m.ChannelID)
}
