package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Nickname            string `json:"nickname"`            // nickname this node will use, besides its IP address
	SharedDirectoryPath string `json:"sharedDirectoryPath"` // the path to the shared directory that is synced among nodes for sharing files.
}

// creates a config file if one doesn't exist yet
func InitializeConfigFile(config Config) {
	if configExists() {
		return
	}
	// create the config file
	configJson, err := json.Marshal(config)
	if err != nil {
		fmt.Println("Failed to marshal json:", err)
		return
	}
	filePath := configFilePath()
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("failed to create config file:", err)
		return
	}
	defer file.Close()

	_, err = file.Write(configJson)
	if err != nil {
		fmt.Println("failed to write config json:", err)
		return
	}
	fmt.Println("Initialized config file:", filePath)
}

func LoadConfig() *Config {
	if !configExists() {
		fmt.Println("failed to load config; config file doesn't exist")
		return nil
	}
	jsonData, err := os.ReadFile(configFilePath())
	if err != nil {
		fmt.Println("failed to read config json file:", err)
		return nil
	}
	var config Config
	err = json.Unmarshal(jsonData, &config)
	if err != nil {
		fmt.Println("error unmarshalling config json:", err)
		return nil
	}
	return &config
}

func SaveConfig(config Config) {
	if !configExists() {
		InitializeConfigFile(config)
		return
	}
	file, err := os.Open(configFilePath())
	if err != nil {
		fmt.Println("failed to open config json:", err)
		return
	}
	defer file.Close()

	jsonData, err := json.Marshal(config)
	if err != nil {
		fmt.Println("failed to marshal config json:", err)
		return
	}
	_, err = file.Write(jsonData)
	if err != nil {
		fmt.Println("failed to write config json:", err)
	}
}

func configExists() bool {
	if _, err := os.Stat(configFilePath()); err == nil {
		return true
	}
	return false
}

func configFilePath() string {
	return filepath.Join("internal", "config", "config.json")
}
