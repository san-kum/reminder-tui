package models

import "time"

type NoteID string

type Priority int

const (
	LowPriority Priority = iota + 1
	MediumPriority
	HighPriority
)

type Note struct {
	ID          NoteID    `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Tags        []string  `json:"tags,omitempty"`
	Priority    Priority  `json:"priority"`
	IsCompleted bool      `json:"is_completed"`
	DueDate     time.Time `json:"due_date,omitempty"`
}

func NewNote(title, content string) *Note {
	now := time.Now()
	return &Note{
		ID:          NoteID(GenerateUniqueID()),
		Title:       title,
		Content:     content,
		CreatedAt:   now,
		UpdatedAt:   now,
		Priority:    MediumPriority,
		IsCompleted: false,
	}
}

func (n *Note) SetDueDate(dueDate time.Time) {
	n.DueDate = dueDate
	n.UpdatedAt = time.Now()
}

func (n *Note) Complete() {
	n.IsCompleted = true
	n.UpdatedAt = time.Now()
}

func (n *Note) Update(title, content string) {
	n.Title = title
	n.Content = content
	n.UpdatedAt = time.Now()
}

func (n *Note) AddTag(tag string) {
	for _, t := range n.Tags {
		if t == tag {
			return
		}
	}
	n.Tags = append(n.Tags, tag)
	n.UpdatedAt = time.Now()
}

func (n *Note) RemoveTag(tag string) {
	for i, t := range n.Tags {
		if t == tag {
			n.Tags = append(n.Tags[:i], n.Tags[i+1:]...)
			n.UpdatedAt = time.Now()
			return
		}
	}
}

func (n *Note) SetPriority(priority Priority) {
	n.Priority = priority
	n.UpdatedAt = time.Now()
}

func GenerateUniqueID() string {
	return time.Now().Format("20060102150405") + RandomString(8)
}

func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)

	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1 * time.Nanosecond)
	}

	return string(result)
}
