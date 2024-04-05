package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/webbben/p2p-file-share/internal/ui"
)

type Config struct {
	Nickname            string `json:"nickname"`            // nickname this node will use, besides its IP address
	SharedDirectoryPath string `json:"sharedDirectoryPath"` // the path to the shared directory that is synced among nodes for sharing files.
}

// creates a config file if one doesn't exist yet
func InitializeConfigFile(config Config) {
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

func GetMountDir(config *Config) string {
	if config != nil {
		return config.SharedDirectoryPath
	}
	conf := LoadConfig()
	return conf.SharedDirectoryPath
}

func SaveConfig(config Config) {
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

func configFilePath() string {
	return filepath.Join("internal", "config", "config.json")
}

// walks the user through creating a new config, and returns it
func NewConfigWorkflow() *Config {
	temp, _ := os.Hostname()
	config := Config{
		Nickname: temp,
	}
	// nickname
	fmt.Printf("Current node nickname: %s", config.Nickname)
	if ui.YorN("Enter a new nickname for this node?") {
		fmt.Print("Nickname: ")
		nickname := ui.ReadInput()
		if nickname == "" {
			fmt.Println("No nickname entered.")
		} else {
			if ui.YorN(fmt.Sprintf("Use nickname \"%s\"?", nickname)) {
				config.Nickname = nickname
			} else {
				fmt.Println("No nickname entered.")
			}
		}
	}
	// fileshare directory
	fmt.Println("Enter a directory path to use for file sharing (Note: it should be empty):")
	directory := ""
	for directory == "" {
		directory = ui.ReadInput()
		if directory == "" {
			fmt.Println("You must enter a valid directory.")
		} else {
			if ui.YorN(fmt.Sprintf("Path: %s.\nAre you sure you want to use this directory?", directory)) {
				config.SharedDirectoryPath = directory
			} else {
				fmt.Println("Gotcha, please enter a different directory then.")
				directory = ""
			}
		}
	}
	return &config
}
