package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/san-kum/reminder-tui/internal/models"
)

type Storage interface {

	// Notes operations
	SaveNote(note *models.Note) error
	GetNote(id models.NoteID) (*models.Note, error)
	GetAllNotes() ([]*models.Note, error)
	DeleteNote(id models.NoteID) error

	// Task operations
	SaveTask(task *models.Task) error
	GetTask(id models.TaskID) (*models.Task, error)
	GetAllTasks() ([]*models.Task, error)
	DeleteTask(id models.TaskID) error

	// Query operations
	GetTasksDueBefore(time time.Time) ([]*models.Task, error)
	GetTasksWithRemindersBy(time time.Time) ([]*models.Task, error)
	GetNotesByTag(tag string) ([]*models.Note, error)
	GetTaskByTag(tag string) ([]*models.Task, error)
}

type FileStorage struct {
	notesFilePath string
	tasksFilePath string
	mutex         sync.RWMutex
}

type notesData struct {
	Notes []*models.Note `json:"notes"`
}

type taskData struct {
	Tasks []*models.Task `json:"tasks"`
}

func NewFileStorage(dataDir string) (*FileStorage, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	return &FileStorage{
		notesFilePath: filepath.Join(dataDir, "notes.json"),
		tasksFilePath: filepath.Join(dataDir, "tasks.json"),
	}, nil
}

func (s *FileStorage) SaveNote(note *models.Note) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	notes, err := s.loadNotes()
	if err != nil {
		return err
	}

	found := false
	for i, n := range notes.Notes {
		if n.ID == note.ID {
			notes.Notes[i] = note
			found = true
			break
		}
	}

	if !found {
		notes.Notes = append(notes.Notes, note)
	}
	return s.saveNotes(notes)

}

func (s *FileStorage) GetNote(id models.NoteID) (*models.Note, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	notes, err := s.loadNotes()
	if err != nil {
		return nil, err
	}
	for _, note := range notes.Notes {
		if note.ID == id {
			return note, nil
		}
	}
	return nil, fmt.Errorf("note with ID %s not found", id)
}

func (s *FileStorage) GetAllNotes() ([]*models.Note, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	notes, err := s.loadNotes()
	if err != nil {
		return nil, err
	}
	return notes.Notes, nil
}

func (s *FileStorage) DeleteNote(id models.NoteID) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	notes, err := s.loadNotes()
	if err != nil {
		return err
	}

	for i, note := range notes.Notes {
		if note.ID == id {
			notes.Notes = append(notes.Notes[:i], notes.Notes[i+1:]...)
			return s.saveNotes(notes)
		}
	}
	return fmt.Errorf("note with ID %s not found.", id)
}

func (s *FileStorage) SaveTask(task *models.Task) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tasks, err := s.loadTasks()
	if err != nil {
		return err
	}

	found := false
	for i, t := range tasks.Tasks {
		if t.ID == task.ID {
			tasks.Tasks[i] = task
			found = true
			break
		}
	}

	if !found {
		tasks.Tasks = append(tasks.Tasks, task)
	}

	return s.saveTasks(tasks)
}

func (s *FileStorage) GetTask(id models.TaskID) (*models.Task, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	tasks, err := s.loadTasks()
	if err != nil {
		return nil, err
	}
	for _, task := range tasks.Tasks {
		if task.ID == id {
			return task, nil
		}
	}
	return nil, fmt.Errorf("task with ID %s not found", err)
}

func (s *FileStorage) GetAllTasks() ([]*models.Task, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	tasks, err := s.loadTasks()
	if err != nil {
		return nil, err
	}
	return tasks.Tasks, nil
}

func (s *FileStorage) DeleteTask(id models.TaskID) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tasks, err := s.loadTasks()
	if err != nil {
		return err
	}
	for i, task := range tasks.Tasks {
		if task.ID == id {
			tasks.Tasks = append(tasks.Tasks[:i], tasks.Tasks[i+1:]...)
			return s.saveTasks(tasks)
		}
	}
	return fmt.Errorf("task with ID %s not found", id)
}

func (s *FileStorage) GetTasksDueBefore(time time.Time) ([]*models.Task, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	allTasks, err := s.loadTasks()
	if err != nil {
		return nil, err
	}
	var result []*models.Task
	for _, task := range allTasks.Tasks {
		if task.DueDate.Before(time) && task.Status != models.TaskStatusCompleted {
			result = append(result, task)
		}
	}
	return result, nil
}

func (s *FileStorage) GetTasksWithRemindersBy(time time.Time) ([]*models.Task, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	allTasks, err := s.loadTasks()
	if err != nil {
		return nil, err
	}
	var result []*models.Task
	for _, task := range allTasks.Tasks {
		if task.ReminderAt.Before(time) && task.Status != models.TaskStatusCompleted {
			result = append(result, task)
		}
	}
	return result, nil
}

func (s *FileStorage) GetNotesByTag(tag string) ([]*models.Note, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	allNotes, err := s.loadNotes()
	if err != nil {
		return nil, err
	}

	var result []*models.Note
	for _, note := range allNotes.Notes {
		for _, noteTag := range note.Tags {
			if noteTag == tag {
				result = append(result, note)
				break
			}
		}
	}
	return result, nil

}

func (s *FileStorage) GetTaskByTag(tag string) ([]*models.Task, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	allTasks, err := s.loadTasks()
	if err != nil {
		return nil, err
	}
	var result []*models.Task
	for _, task := range allTasks.Tasks {
		for _, taskTag := range task.Tags {
			if taskTag == tag {
				result = append(result, task)
				break
			}
		}
	}
	return result, nil
}

func (s *FileStorage) loadNotes() (*notesData, error) {
	notes := &notesData{
		Notes: []*models.Note{},
	}

	if _, err := os.Stat(s.notesFilePath); os.IsNotExist(err) {
		return notes, s.saveNotes(notes)
	}

	// Read the file
	data, err := os.ReadFile(s.notesFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read notes: %w", err)
	}

	// Parse JSON
	if err := json.Unmarshal(data, notes); err != nil {
		return nil, fmt.Errorf("failed to parse notes file: %w", err)
	}
	return notes, nil
}

func (s *FileStorage) saveNotes(notes *notesData) error {
	data, err := json.MarshalIndent(notes, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal notes data: %w", err)
	}

	if err := os.WriteFile(s.notesFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write notes file: %w", err)
	}
	return nil
}

func (s *FileStorage) loadTasks() (*taskData, error) {
	tasks := &taskData{
		Tasks: []*models.Task{},
	}

	if _, err := os.Stat(s.tasksFilePath); os.IsNotExist(err) {
		return tasks, s.saveTasks(tasks)
	}

	data, err := os.ReadFile(s.tasksFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tasks file: %w", err)
	}

	if err := json.Unmarshal(data, tasks); err != nil {
		return nil, fmt.Errorf("failed to parse tasks file: %w", err)
	}
	return tasks, nil
}

func (s *FileStorage) saveTasks(tasks *taskData) error {
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks data: %w", err)
	}

	if err := os.WriteFile(s.tasksFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write tasks: %w", err)
	}

	return nil
}
