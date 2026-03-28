package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stashcut/cli/config"
	"github.com/stashcut/cli/store"
	"github.com/stashcut/cli/ui"
)

func main() {
	filePath := flag.String("file", "", "Path to shortcuts.json (overrides config)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	dataPath := cfg.DataFilePath
	if *filePath != "" {
		dataPath = *filePath
	}

	data, err := store.Load(dataPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading shortcuts file: %v\n", err)
		os.Exit(1)
	}

	if data == nil {
		// First run: show TUI prompt
		defaultPath, _ := config.DefaultDataPath()
		frModel := ui.NewFirstRunModel(defaultPath)
		p := tea.NewProgram(frModel)
		result, err := p.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fr, ok := result.(ui.FirstRunModel)
		if !ok || fr.Result == nil {
			// User quit without choosing
			os.Exit(0)
		}

		dataPath = fr.Result.Path
		if fr.Result.Create {
			data = store.New()
			if err := store.Save(dataPath, data); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating shortcuts file: %v\n", err)
				os.Exit(1)
			}
		} else {
			// User specified an existing file path
			data, err = store.Load(dataPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading file: %v\n", err)
				os.Exit(1)
			}
			if data == nil {
				// File doesn't exist yet at specified path; create it
				data = store.New()
				if err := store.Save(dataPath, data); err != nil {
					fmt.Fprintf(os.Stderr, "Error creating shortcuts file: %v\n", err)
					os.Exit(1)
				}
			}
		}

		// Persist the chosen path to config
		cfg.DataFilePath = dataPath
		config.Save(cfg) //nolint
	}

	appModel := ui.NewAppModel(data, dataPath)
	p := tea.NewProgram(appModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running app: %v\n", err)
		os.Exit(1)
	}
}
