package ui

import (
	"fmt"
	"math"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/san-kum/reminder-tui/internal/models"
	"github.com/san-kum/reminder-tui/internal/storage"
)

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	helpStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render
)

type NotesApp struct {
	storage       storage.Storage
	notesList     list.Model
	tasksList     list.Model
	activeView    string
	err           error
	activeInput   int
	inputs        []textinput.Model
	creating      bool
	creatingTask  bool
	editing       bool
	selectedNote  *models.Note
	selectedTask  *models.Task
	width, height int
}

type noteItem struct {
	note *models.Note
}

func (i noteItem) Title() string {
	status := " "
	if i.note.IsCompleted {
		status = "✓"
	}
	return fmt.Sprintf("[%s] %s", status, i.note.Title)
}

func (i noteItem) Description() string {
	return fmt.Sprintf("Created: %s", i.note.CreatedAt.Format("Jan 2, 2006"))
}

func (i noteItem) FilterValue() string { return i.note.Title }

type taskItem struct {
	task *models.Task
}

func (i taskItem) Title() string {
	var status string
	switch i.task.Status {
	case models.TaskStatusCompleted:
		status = "✓"
	case models.TaskStatusOverdue:
		status = "!"
	case models.TaskStatusInProgress:
		status = "►"
	default:
		status = " "
	}
	return fmt.Sprintf("[%s] %s", status, i.task.Title)
}

func (i taskItem) Description() string {
	return fmt.Sprintf("Due: %s", i.task.DueDate.Format("Jan 2, 2006 at 3:04 PM"))
}

func (i taskItem) FilterValue() string { return i.task.Title }

func NewNotesApp(s storage.Storage) *NotesApp {
	// Set up note list
	noteDelegate := list.NewDefaultDelegate()
	noteItems := []list.Item{}
	notesList := list.New(noteItems, noteDelegate, 0, 0)
	notesList.Title = "Notes"
	notesList.SetShowHelp(false)

	// Set up task list
	taskDelegate := list.NewDefaultDelegate()
	taskItems := []list.Item{}
	tasksList := list.New(taskItems, taskDelegate, 0, 0)
	tasksList.Title = "Tasks"
	tasksList.SetShowHelp(false)

	// Initialize inputs for creating/editing notes and tasks
	inputs := make([]textinput.Model, 4)
	for i := range inputs {
		t := textinput.New()
		t.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
		t.CharLimit = 100

		switch i {
		case 0:
			t.Placeholder = "Title"
			t.Focus()
		case 1:
			t.Placeholder = "Content/Description"
			t.CharLimit = 500
		case 2:
			t.Placeholder = "Due Date (YYYY-MM-DD)"
		case 3:
			t.Placeholder = "Reminder (e.g., 1h, 30m, 1d before due date)"
		}

		inputs[i] = t
	}

	return &NotesApp{
		storage:      s,
		notesList:    notesList,
		tasksList:    tasksList,
		activeView:   "notes",
		inputs:       inputs,
		activeInput:  0,
		creating:     false,
		creatingTask: false,
		editing:      false,
	}
}

func (m *NotesApp) Init() tea.Cmd {
	// Load initial data
	return tea.Batch(
		m.loadNotes(),
		m.loadTasks(),
	)
}

func (m *NotesApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global keys
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "tab":
			if !m.creating && !m.editing {
				// Toggle between notes and tasks
				if m.activeView == "notes" {
					m.activeView = "tasks"
				} else {
					m.activeView = "notes"
				}
			}
			return m, nil

		case "n":
			if !m.creating && !m.editing {
				// Start creating a new note/task
				m.creating = true
				m.creatingTask = m.activeView == "tasks"
				m.resetInputs()
				m.inputs[0].Focus()
				m.activeInput = 0
				return m, nil
			}

		case "e":
			if !m.creating && !m.editing {
				// Start editing the selected note/task
				if m.activeView == "notes" && m.selectedNote != nil {
					m.editing = true
					m.inputs[0].SetValue(m.selectedNote.Title)
					m.inputs[1].SetValue(m.selectedNote.Content)
					m.inputs[0].Focus()
					m.activeInput = 0
				} else if m.activeView == "tasks" && m.selectedTask != nil {
					m.editing = true
					m.creatingTask = true
					m.inputs[0].SetValue(m.selectedTask.Title)
					m.inputs[1].SetValue(m.selectedTask.Description)
					m.inputs[2].SetValue(m.selectedTask.DueDate.Format("2006-01-02"))
					reminderPeriod := m.selectedTask.DueDate.Sub(m.selectedTask.ReminderAt)
					m.inputs[3].SetValue(formatDuration(reminderPeriod))
					m.inputs[0].Focus()
					m.activeInput = 0
				}
				return m, nil
			}

		case "d":
			if !m.creating && !m.editing {
				// Delete the selected note/task
				if m.activeView == "notes" && m.selectedNote != nil {
					return m, tea.Batch(
						m.deleteNote(m.selectedNote.ID),
						m.loadNotes(),
					)
				} else if m.activeView == "tasks" && m.selectedTask != nil {
					return m, tea.Batch(
						m.deleteTask(m.selectedTask.ID),
						m.loadTasks(),
					)
				}
			}

		case "c":
			if !m.creating && !m.editing {
				// Toggle completion status
				if m.activeView == "notes" && m.selectedNote != nil {
					m.selectedNote.IsCompleted = !m.selectedNote.IsCompleted
					return m, tea.Batch(
						m.saveNote(m.selectedNote),
						m.loadNotes(),
					)
				} else if m.activeView == "tasks" && m.selectedTask != nil {
					if m.selectedTask.Status == models.TaskStatusCompleted {
						m.selectedTask.Status = models.TaskStatusPending
					} else {
						m.selectedTask.Complete()
					}
					return m, tea.Batch(
						m.saveTask(m.selectedTask),
						m.loadTasks(),
					)
				}
			}
		}

		// Handle inputs while creating/editing
		if m.creating || m.editing {
			switch msg.String() {
			case "esc":
				// Cancel creating/editing
				m.creating = false
				m.editing = false
				m.creatingTask = false
				return m, nil

			case "enter":
				if m.activeInput == len(m.inputs)-1 ||
					(!m.creatingTask && m.activeInput == 1) {
					// Submit the form
					return m, m.handleFormSubmit()
				}
				// Move to the next input
				m.nextInput()
				return m, nil

			case "tab", "shift+tab":
				// Navigate between inputs
				if msg.String() == "tab" {
					m.nextInput()
				} else {
					m.prevInput()
				}
				return m, nil
			}

			// Handle input changes
			cmd := m.updateInputs(msg)
			return m, cmd
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.notesList.SetSize(msg.Width/2-2, msg.Height-10)
		m.tasksList.SetSize(msg.Width/2-2, msg.Height-10)
		return m, nil
	}

	// Handle list updates
	var cmd tea.Cmd
	if m.activeView == "notes" {
		m.notesList, cmd = m.notesList.Update(msg)
		cmds = append(cmds, cmd)

		// Update selected note
		if i, ok := m.notesList.SelectedItem().(noteItem); ok {
			m.selectedNote = i.note
		}
	} else {
		m.tasksList, cmd = m.tasksList.Update(msg)
		cmds = append(cmds, cmd)

		// Update selected task
		if i, ok := m.tasksList.SelectedItem().(taskItem); ok {
			m.selectedTask = i.task
		}
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m *NotesApp) View() string {
	if m.creating || m.editing {
		return m.formView()
	}

	var view string

	// Header
	titleText := "Notes & Tasks CLI"
	view = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170")).
		Render(titleText) + "\n\n"

	// Content
	var content string
	if m.activeView == "notes" {
		notesList := m.notesList.View()

		// Detail view for selected note
		detailView := "Select a note to view details"
		if m.selectedNote != nil {
			detailView = fmt.Sprintf(
				"Title: %s\n\nContent:\n%s\n\nCreated: %s\nUpdated: %s\n\nTags: %v\n\nStatus: %s",
				m.selectedNote.Title,
				m.selectedNote.Content,
				m.selectedNote.CreatedAt.Format("Jan 2, 2006 15:04"),
				m.selectedNote.UpdatedAt.Format("Jan 2, 2006 15:04"),
				m.selectedNote.Tags,
				func() string {
					if m.selectedNote.IsCompleted {
						return "Completed"
					}
					return "Pending"
				}(),
			)
		}

		// Split view with notes list on the left and details on the right
		notesPanel := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1).
			Width(m.width/2 - 4).
			Render(notesList)

		detailPanel := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1).
			Width(m.width/2 - 4).
			Render(detailView)

		content = lipgloss.JoinHorizontal(lipgloss.Top, notesPanel, detailPanel)
	} else {
		tasksList := m.tasksList.View()

		// Detail view for selected task
		detailView := "Select a task to view details"
		if m.selectedTask != nil {
			detailView = fmt.Sprintf(
				"Title: %s\n\nDescription:\n%s\n\nDue: %s\nReminder: %s\n\nStatus: %s\nPriority: %s\n\nTags: %v",
				m.selectedTask.Title,
				m.selectedTask.Description,
				m.selectedTask.DueDate.Format("Jan 2, 2006 15:04"),
				m.selectedTask.ReminderAt.Format("Jan 2, 2006 15:04"),
				func() string {
					switch m.selectedTask.Status {
					case models.TaskStatusCompleted:
						return "Completed"
					case models.TaskStatusInProgress:
						return "In Progress"
					case models.TaskStatusOverdue:
						return "Overdue"
					default:
						return "Pending"
					}
				}(),
				func() string {
					switch m.selectedTask.Priority {
					case models.LowPriority:
						return "Low"
					case models.MediumPriority:
						return "Medium"
					case models.HighPriority:
						return "High"
					default:
						return "Unknown"
					}
				}(),
				m.selectedTask.Tags,
			)
		}

		// Split view with tasks list on the left and details on the right
		tasksPanel := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1).
			Width(m.width/2 - 4).
			Render(tasksList)

		detailPanel := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1).
			Width(m.width/2 - 4).
			Render(detailView)

		content = lipgloss.JoinHorizontal(lipgloss.Top, tasksPanel, detailPanel)
	}

	view += content + "\n\n"

	// Help text at the bottom
	var help string
	if m.activeView == "notes" {
		help = helpStyle("tab: switch to tasks • n: new note • e: edit note • d: delete note • c: toggle completion • q: quit")
	} else {
		help = helpStyle("tab: switch to notes • n: new task • e: edit task • d: delete task • c: toggle completion • q: quit")
	}

	view += help

	return view
}

// formView displays the form for creating or editing notes and tasks
func (m *NotesApp) formView() string {
	var title string
	if m.creating {
		if m.creatingTask {
			title = "Create New Task"
		} else {
			title = "Create New Note"
		}
	} else {
		if m.creatingTask {
			title = "Edit Task"
		} else {
			title = "Edit Note"
		}
	}

	var form string
	form = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170")).
		Render(title) + "\n\n"

	// Add inputs
	for i := range m.inputs {
		if !m.creatingTask && i > 1 {
			continue // Only show title and content for notes
		}

		field := m.inputs[i].View()
		form += field + "\n"
	}

	form += "\n" + helpStyle("enter: submit • tab: next field • esc: cancel")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1).
		Width(m.width - 4).
		Render(form)
}

// Helper methods

// nextInput focuses the next input field
func (m *NotesApp) nextInput() {
	m.inputs[m.activeInput].Blur()
	m.activeInput = (m.activeInput + 1) % len(m.inputs)
	if !m.creatingTask && m.activeInput > 1 {
		m.activeInput = 0 // Cycle back for notes (only title and content)
	}
	m.inputs[m.activeInput].Focus()
}

// prevInput focuses the previous input field
func (m *NotesApp) prevInput() {
	m.inputs[m.activeInput].Blur()
	m.activeInput--
	if m.activeInput < 0 {
		if m.creatingTask {
			m.activeInput = len(m.inputs) - 1
		} else {
			m.activeInput = 1 // For notes, only go back to content field
		}
	}
	m.inputs[m.activeInput].Focus()
}

// resetInputs clears all input fields
func (m *NotesApp) resetInputs() {
	for i := range m.inputs {
		m.inputs[i].SetValue("")
	}
}

// updateInputs handles input updates
func (m *NotesApp) updateInputs(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	// Only update the active input
	m.inputs[m.activeInput], cmd = m.inputs[m.activeInput].Update(msg)

	return cmd
}

// handleFormSubmit processes the form submission
func (m *NotesApp) handleFormSubmit() tea.Cmd {
	if m.creatingTask {
		// Create or edit task
		title := m.inputs[0].Value()
		description := m.inputs[1].Value()
		dueDateStr := m.inputs[2].Value()
		reminderStr := m.inputs[3].Value()

		// Validate inputs
		if title == "" {
			return nil // Ignore empty title
		}

		// Parse due date
		dueDate, err := time.Parse("2006-01-02", dueDateStr)
		if err != nil {
			// Default to tomorrow if not valid
			dueDate = time.Now().Add(24 * time.Hour)
		}

		// Parse reminder period
		reminderPeriod, err := parseDuration(reminderStr)
		if err != nil {
			// Default to 1 hour before if not valid
			reminderPeriod = 1 * time.Hour
		}

		if m.editing && m.selectedTask != nil {
			// Update existing task
			m.selectedTask.Update(title, description, dueDate)
			m.selectedTask.SetReminderPeriod(reminderPeriod)

			m.editing = false
			m.creatingTask = false
			m.resetInputs()

			return tea.Batch(
				m.saveTask(m.selectedTask),
				m.loadTasks(),
			)
		} else {
			// Create new task
			task := models.NewTask(title, description, dueDate)
			task.SetReminderPeriod(reminderPeriod)

			m.creating = false
			m.creatingTask = false
			m.resetInputs()

			return tea.Batch(
				m.saveTask(task),
				m.loadTasks(),
			)
		}
	} else {
		// Create or edit note
		title := m.inputs[0].Value()
		content := m.inputs[1].Value()

		// Validate inputs
		if title == "" {
			return nil // Ignore empty title
		}

		if m.editing && m.selectedNote != nil {
			// Update existing note
			m.selectedNote.Update(title, content)

			m.editing = false
			m.resetInputs()

			return tea.Batch(
				m.saveNote(m.selectedNote),
				m.loadNotes(),
			)
		} else {
			// Create new note
			note := models.NewNote(title, content)

			m.creating = false
			m.resetInputs()

			return tea.Batch(
				m.saveNote(note),
				m.loadNotes(),
			)
		}
	}
}

// loadNotes loads notes from storage
func (m *NotesApp) loadNotes() tea.Cmd {
	return func() tea.Msg {
		notes, err := m.storage.GetAllNotes()
		if err != nil {
			// Handle error
			return nil
		}

		// Convert to list items
		items := make([]list.Item, len(notes))
		for i, note := range notes {
			items[i] = noteItem{note: note}
		}

		// Update the list
		m.notesList.SetItems(items)

		return nil
	}
}

// loadTasks loads tasks from storage
func (m *NotesApp) loadTasks() tea.Cmd {
	return func() tea.Msg {
		tasks, err := m.storage.GetAllTasks()
		if err != nil {
			// Handle error
			return nil
		}

		// Convert to list items
		items := make([]list.Item, len(tasks))
		for i, task := range tasks {
			items[i] = taskItem{task: task}
		}

		// Update the list
		m.tasksList.SetItems(items)

		return nil
	}
}

// saveNote saves a note to storage
func (m *NotesApp) saveNote(note *models.Note) tea.Cmd {
	return func() tea.Msg {
		err := m.storage.SaveNote(note)
		if err != nil {
			// Handle error
			return nil
		}
		return nil
	}
}

// saveTask saves a task to storage
func (m *NotesApp) saveTask(task *models.Task) tea.Cmd {
	return func() tea.Msg {
		err := m.storage.SaveTask(task)
		if err != nil {
			// Handle error
			return nil
		}
		return nil
	}
}

// deleteNote deletes a note from storage
func (m *NotesApp) deleteNote(id models.NoteID) tea.Cmd {
	return func() tea.Msg {
		err := m.storage.DeleteNote(id)
		if err != nil {
			// Handle error
			return nil
		}
		m.selectedNote = nil
		return nil
	}
}

// deleteTask deletes a task from storage
func (m *NotesApp) deleteTask(id models.TaskID) tea.Cmd {
	return func() tea.Msg {
		err := m.storage.DeleteTask(id)
		if err != nil {
			return nil
		}
		m.selectedTask = nil
		return nil
	}
}

func parseDuration(s string) (time.Duration, error) {
	if len(s) > 0 && s[len(s)-1] == 'd' {
		days, err := fmt.Sscanf(s, "%dd", new(int))
		if err == nil && days > 0 {
			return time.Duration(days) * 24 * time.Hour, nil
		}
	}

	return time.ParseDuration(s)
}

func formatDuration(d time.Duration) string {
	hours := d.Hours()
	if hours >= 24 && math.Mod(hours, 24) == 0 {
		return fmt.Sprintf("%dd", int(hours/24))
	}

	return d.String()
}
