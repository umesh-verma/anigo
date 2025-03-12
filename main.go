package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/umesh-verma/anigo/sources"
	"github.com/umesh-verma/anigo/sources/wpanime"
	"github.com/umesh-verma/anigo/tui"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Sources map[string]struct {
		Name       string `yaml:"name"`
		Type       string `yaml:"type"`
		BaseURL    string `yaml:"base_url"`
		SearchPath string `yaml:"search_path"`
	} `yaml:"sources"`
}

func main() {
	// Load config
	config := loadConfig()

	// Initialize sources
	sources := make(map[string]sources.SourceProvider)
	for id, src := range config.Sources {
		switch src.Type {
		case "wp-anime":
			sources[id] = wpanime.New(src.BaseURL, src.SearchPath)
		}
	}

	// Start TUI
	p := tea.NewProgram(tui.New(sources), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func loadConfig() Config {
	data, err := os.ReadFile("config/sources.yaml")
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		os.Exit(1)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		fmt.Printf("Error parsing config: %v\n", err)
		os.Exit(1)
	}

	return config
}
