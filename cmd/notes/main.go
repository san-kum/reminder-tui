package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/san-kum/reminder-tui/internal/reminder"
	"github.com/san-kum/reminder-tui/internal/storage"
	"github.com/san-kum/reminder-tui/internal/ui"
)

func main() {
	var dataDir string

	homeDir, err := os.UserHomeDir()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}
	defaultDataDir := filepath.Join(homeDir, ".cli-notes")
	flag.StringVar(&dataDir, "data", defaultDataDir, "Directory to store notes and and tasks data")
	flag.Parse()

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating data directory: %v\n", err)
		os.Exit(1)
	}
	s, err := storage.NewFileStorage(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing storage: %v\n", err)
		os.Exit(1)
	}

	notifier := &reminder.ConsoleNotifier{}
	reminderService := reminder.NewReminderService(s, notifier, 1*time.Minute)

	reminderService.Start()
	defer reminderService.Stop()

	app := ui.NewNotesApp(s)

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
		os.Exit(1)
	}

}
