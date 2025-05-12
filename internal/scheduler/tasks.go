package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Al-Pragliola/poc-dev-agents/internal/agent"
	"github.com/google/uuid"
)

type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
)

type Task struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	AssignedTo  string     `json:"assigned_to"`
	Status      TaskStatus `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type TaskScheduler struct {
	ctx          context.Context
	Tasks        []*Task
	Agents       map[string]*agent.Agent
	ToolCaller   *ToolCaller
	OutputFolder string
}

func NewTaskScheduler(agents map[string]*agent.Agent, outputFolder string) *TaskScheduler {
	t := &TaskScheduler{
		ctx:          context.Background(),
		Tasks:        []*Task{},
		Agents:       agents,
		OutputFolder: outputFolder,
	}

	t.ToolCaller = NewToolCaller(t)

	return t
}

func (t *TaskScheduler) Run() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			slog.Info("Stopping task scheduler...")
			return
		case <-ticker.C:

			if t.checkForInProgressTasks() {
				slog.Debug("There are still tasks in progress, skipping...")

				continue
			}

			// Get the next pending task
			task := t.getNextPendingTask()

			if task == nil {
				slog.Debug("No pending tasks, skipping...")

				continue
			}

			slog.Info("Scheduling task", "task", task.Description, "to", task.AssignedTo)

			// Update task status to in progress
			if err := t.updateTaskStatus(task, TaskStatusInProgress); err != nil {
				slog.Error("Failed to update task status", "task", task.Description, "error", err)

				continue
			}

			// Execute the task
			if err := t.executeTask(task.AssignedTo, task); err != nil {
				slog.Error("Task execution failed", "task", task.Description, "error", err)

				if err := t.updateTaskStatus(task, TaskStatusFailed); err != nil {
					slog.Error("Failed to update task status to failed", "task", task.Description, "error", err)
				}

				return
			}
		}
	}
}

func (t *TaskScheduler) AddTask(task *Task) {
	task.ID = uuid.New().String()
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	task.Status = TaskStatusPending

	slog.Info("Adding task", "task", task.Description, "assigned to", task.AssignedTo)

	t.Tasks = append(t.Tasks, task)
}

func (t *TaskScheduler) GetTask(id string) *Task {
	for _, task := range t.Tasks {
		if task.ID == id {
			return task
		}
	}

	return nil
}

func (t *TaskScheduler) GetTasks() []*Task {
	return t.Tasks
}

func (t *TaskScheduler) GetTasksByAssignedTo(assignedTo string) []*Task {
	var tasks []*Task

	for _, task := range t.Tasks {
		if task.AssignedTo == assignedTo {
			tasks = append(tasks, task)
		}
	}

	return tasks
}

func (t *TaskScheduler) Stop() {
	t.ctx.Done()
}

func (t *TaskScheduler) executeTask(agentName string, task *Task) error {
	agent := t.Agents[agentName]

	if agent == nil {
		return fmt.Errorf("agent not found")
	}

	slog.Info("Executing task", "task", task.Description, "assigned to", agentName)

	resp, err := agent.Chat(t.ctx, task.Description)
	if err != nil {
		return fmt.Errorf("error executing task: %w", err)
	}

	if resp.Message != "" {
		slog.Debug("The response from the agent is: ", "response", resp.Message)
	}

	if len(resp.ToolsCalls) > 0 {
		slog.Debug("The tools calls from the agent are: ", "toolsCalls", resp.ToolsCalls)

		for _, toolCall := range resp.ToolsCalls {
			_, err := t.ToolCaller.Call(toolCall.Function.Name, toolCall.Function.Arguments)
			if err != nil {
				slog.Error("Error calling tool:", "error", err)

				return fmt.Errorf("error calling tool: %w", err)
			}
		}
	}

	if err := t.updateTaskStatus(task, TaskStatusCompleted); err != nil {
		slog.Error("Failed to update task status to completed", "task", task.Description, "error", err)

		return fmt.Errorf("failed to update task status to completed: %w", err)
	}

	return nil
}

func (t *TaskScheduler) checkForInProgressTasks() bool {
	for _, task := range t.Tasks {
		if task.Status == TaskStatusInProgress {
			return true
		}
	}

	return false
}

func (t *TaskScheduler) getNextPendingTask() *Task {
	for _, task := range t.Tasks {
		if task.Status == TaskStatusPending {
			return task
		}
	}

	return nil
}

func (t *TaskScheduler) updateTaskStatus(task *Task, status TaskStatus) error {
	task.Status = status
	task.UpdatedAt = time.Now()

	return nil
}
