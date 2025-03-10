package models

import (
	"time"
)

type TaskID string

type TaskStatus int

const (
	TaskStatusPending TaskStatus = iota
	TaskStatusInProgress
	TaskStatusCompleted
	TaskStatusOverdue
)

type Task struct {
	ID          TaskID     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DueDate     time.Time  `json:"due_date"`
	ReminderAt  time.Time  `json:"reminder_at"`
	Priority    Priority   `json:"priority"`
	Status      TaskStatus `json:"status"`
	Tags        []string   `json:"tags,omitempty"`
	NoteID      NoteID     `json:"note_id,omitempty"`
}

func NewTask(title, description string, dueDate time.Time) *Task {
	now := time.Now()

	reminderAt := dueDate.Add(-1 * time.Hour)

	return &Task{
		ID:          TaskID(GenerateUniqueID()),
		Title:       title,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
		DueDate:     dueDate,
		ReminderAt:  reminderAt,
		Priority:    MediumPriority,
		Status:      TaskStatusPending,
	}
}

func (t *Task) SetReminderTime(reminderAt time.Time) {
	t.ReminderAt = reminderAt
	t.UpdatedAt = time.Now()
}

func (t *Task) SetReminderPeriod(period time.Duration) {
	t.ReminderAt = t.DueDate.Add(-period)
	t.UpdatedAt = time.Now()
}

func (t *Task) MarkInProgress() {
	t.Status = TaskStatusCompleted
	t.UpdatedAt = time.Now()
}

func (t *Task) Complete() {
	t.Status = TaskStatusCompleted
	t.UpdatedAt = time.Now()
}

func (t *Task) Update(title, description string, dueDate time.Time) {
	t.Title = title
	t.Description = description
	t.DueDate = dueDate
	t.UpdatedAt = time.Now()

	offset := t.DueDate.Sub(t.ReminderAt)
	t.ReminderAt = dueDate.Add(-offset)
}

func (t *Task) IsOverDue() bool {
	return time.Now().After(t.DueDate) && t.Status != TaskStatusCompleted
}

func (t *Task) UpdateStatus() {
	if t.Status == TaskStatusCompleted {
		return
	}

	if t.IsOverDue() {
		t.Status = TaskStatusOverdue
	}
}


func (t *Task) AddTag(tag string){
  for _, existingTag := range t.Tags{
    if existingTag == tag {
      return
    }
  }
  t.Tags = append(t.Tags, tag)
  t.UpdatedAt = time.Now()
}

func (t* Task) RemoveTag(tag string){
  for i, existingTag := range t.Tags{
    if existingTag == tag {
      t.Tags = append(t.Tags, t.Tags[i+1:]...)
      t.UpdatedAt = time.Now()
      return
    }
  }
}



func (t *Task) SetPriority(priority Priority){
  t.Priority = priority
  t.UpdatedAt = time.Now()
}

func (t *Task) LinkToNote(noteID NoteID){
  t.NoteID = noteID
  t.UpdatedAt = time.Now()
}



