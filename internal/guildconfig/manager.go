package guildconfig

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"
)

const defaultConfigFile = "guild_configs.json"

type GuildSetting struct {
	GuildID      string `json:"guild_id"`
	LLMChannelID string `json:"llm_channel_id"`
}

type Manager struct {
	configs  map[string]*GuildSetting // Key: GuildID
	mu       sync.RWMutex
	filePath string
}

func NewManager(configPath string) (*Manager, error) {
	if configPath == "" {
		execPath, err := os.Executable()
		if err != nil {
			log.Printf("⚠️: Could not get executable path: %v\n", err)
			configPath = "."
		} else {
			configPath = filepath.Dir(execPath)
		}
	}
	filePath := filepath.Join(configPath, defaultConfigFile)
	m := &Manager{
		configs:  make(map[string]*GuildSetting),
		filePath: filePath,
	}
	if err := m.load(); err != nil {
		if _, ok := err.(*fs.PathError); !ok && err.Error() != "EOF" {
			log.Printf("Warning: Could not load guild configs from %s: %v. Starting with empty configs.", filePath, err)
		} else {
			log.Printf("No guild configs found at %s or file is empty. Starting fresh.", filePath)
		}
	}
	return m, nil

}

func (m *Manager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return err
	}

	if len(data) == 0 { // Empty file
		m.configs = make(map[string]*GuildSetting)
		return nil
	}

	var settingsList []*GuildSetting
	if err := json.Unmarshal(data, &settingsList); err != nil {
		return fmt.Errorf("error unmarshalling guild configs: %w", err)
	}

	m.configs = make(map[string]*GuildSetting)
	for _, setting := range settingsList {
		m.configs[setting.GuildID] = setting
	}
	log.Printf("Loaded %d guild configurations from %s", len(m.configs), m.filePath)
	return nil
}

func (m *Manager) save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var settingsList []*GuildSetting
	for _, setting := range m.configs {
		settingsList = append(settingsList, setting)
	}

	data, err := json.MarshalIndent(settingsList, "", "  ")
	if err != nil {
		return fmt.Errorf("Error marshalling guild configs: %w", err)
	}

	dir := filepath.Dir(m.filePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if mkDirErr := os.MkdirAll(dir, 0750); mkDirErr != nil {
			return fmt.Errorf("failed to create config directory %s: %w", dir, mkDirErr)
		}
	}

	return os.WriteFile(m.filePath, data, 0640)
}

func (m *Manager) GetLLMChannel(guildID string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	setting, found := m.configs[guildID]
	if !found || setting.LLMChannelID == "" {
		return "", false
	}
	return setting.LLMChannelID, true
}

func (m *Manager) SetLLMChannel(guildID string, channelID string) error {
	m.mu.Lock()
	setting, found := m.configs[guildID]
	if !found {
		setting = &GuildSetting{GuildID: guildID}
		m.configs[guildID] = setting
	}
	setting.LLMChannelID = channelID
	m.mu.Unlock()

	if err := m.save(); err != nil {
		return fmt.Errorf("failed to save guild configs after setting channel for %s: %w", guildID, err)
	}
	log.Printf("Set LLM channel for guild %s to %s", guildID, channelID)
	return nil
}

func (m *Manager) RemoveLLMChannel(guildID string) error {
	m.mu.Lock()
	_, found := m.configs[guildID]
	if !found {
		m.mu.Unlock()
		return nil
	}

	m.configs[guildID].LLMChannelID = ""
	m.mu.Unlock()

	if err := m.save(); err != nil {
		return fmt.Errorf("failed to save guild configs after removing channel for %s: %w", guildID, err)
	}
	log.Printf("Removed LLM channel configuration for guild %s", guildID)
	return nil
}
