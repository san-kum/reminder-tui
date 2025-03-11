package reminder

import (
	"fmt"
	"sync"
	"time"

	"github.com/san-kum/reminder-tui/internal/models"
	"github.com/san-kum/reminder-tui/internal/storage"
)

type Notifier interface {
	Notify(task *models.Task) error
}

type ConsoleNotifier struct{}

func (n *ConsoleNotifier) Notify(task *models.Task) error {
	fmt.Printf("\n[REMINDER] Task: %s is due on %s\n", task.Title, task.DueDate.Format("Jan 2, 2006 at 3:04 PM"))
	return nil
}

type ReminderService struct {
	storage        storage.Storage
	notifier       Notifier
	checkInterval  time.Duration
	stopChan       chan struct{}
	wg             sync.WaitGroup
	remindersMutex sync.Mutex
	sentReminders  map[models.TaskID]time.Time
}

func NewReminderService(storage storage.Storage, notifier Notifier, checkInterval time.Duration) *ReminderService {
	return &ReminderService{
		storage:       storage,
		notifier:      notifier,
		checkInterval: checkInterval,
		stopChan:      make(chan struct{}),
		sentReminders: make(map[models.TaskID]time.Time),
	}
}

func (r *ReminderService) Start() {
	r.wg.Add(1)
	go r.reminderLoop()
}

func (r *ReminderService) Stop() {
	close(r.stopChan)
	r.wg.Wait()
}

func (r *ReminderService) reminderLoop() {
	defer r.wg.Done()

	ticker := time.NewTicker(r.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.checkReminders()
		case <-r.stopChan:
			return
		}
	}
}

func (r *ReminderService) checkReminders() {
	now := time.Now()
	tasks, err := r.storage.GetTasksWithRemindersBy(now)
	if err != nil {
		fmt.Printf("error checking reminders %v\n", err)
		return
	}

	for _, task := range tasks {
		r.remindersMutex.Lock()
		lastSent, found := r.sentReminders[task.ID]
		shouldSend := !found || now.Sub(lastSent) > 6*time.Hour
		if shouldSend {
			r.sentReminders[task.ID] = now
			r.remindersMutex.Unlock()

			task.UpdateStatus()
			r.storage.SaveTask(task)

			r.notifier.Notify(task)
		} else {
			r.remindersMutex.Unlock()
		}
	}

	r.remindersMutex.Lock()
	for id, sentTime := range r.sentReminders {
		if now.Sub(sentTime) > 24*time.Hour {
			delete(r.sentReminders, id)
		}
	}
	r.remindersMutex.Unlock()

}

func (r *ReminderService) CreateTaskWithReminder(title, description string, dueDate time.Time, reminderPeriod time.Duration) (*models.Task, error) {
	task := models.NewTask(title, description, dueDate)
	task.SetReminderPeriod(reminderPeriod)
	if err := r.storage.SaveTask(task); err != nil {
		return nil, err
	}
	return task, nil
}
