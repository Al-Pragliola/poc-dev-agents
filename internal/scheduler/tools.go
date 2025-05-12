package scheduler

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ToolCaller struct {
	Tools         map[string]func(args map[string]any) (string, error)
	TaskScheduler *TaskScheduler
}

func NewToolCaller(taskScheduler *TaskScheduler) *ToolCaller {
	t := &ToolCaller{
		TaskScheduler: taskScheduler,
	}

	toolsFuncMap := map[string]func(args map[string]any) (string, error){
		"assign-task": func(args map[string]any) (string, error) {
			return t.assignTask(args)
		},
		"run-command": func(args map[string]any) (string, error) {
			return t.runCommand(args)
		},
		"write-file": func(args map[string]any) (string, error) {
			return t.writeFile(args)
		},
		"read-file": func(args map[string]any) (string, error) {
			return t.readFile(args)
		},
		"list-files": func(args map[string]any) (string, error) {
			return t.listFiles(args)
		},
		"edit-file": func(args map[string]any) (string, error) {
			return t.editFile(args)
		},
	}

	t.Tools = toolsFuncMap

	return t
}

func (t *ToolCaller) Call(toolName string, args map[string]any) (string, error) {
	if _, ok := t.Tools[toolName]; !ok {
		return "", fmt.Errorf("tool %s not found", toolName)
	}

	return t.Tools[toolName](args)
}

func (t *ToolCaller) assignTask(args map[string]any) (string, error) {
	assignee := args["assignee"].(string)
	taskDescription := args["task"].(string)

	task := &Task{
		Description: taskDescription,
		AssignedTo:  assignee,
	}

	t.TaskScheduler.AddTask(task)

	return "", nil
}

func (t *ToolCaller) runCommand(args map[string]any) (string, error) {
	workingDirectory := t.TaskScheduler.OutputFolder

	if args["command"] == nil {
		return "", fmt.Errorf("command is required")
	}

	command := args["command"].(string)

	if args["working_directory"] != nil {
		requestedWorkingDirectory := args["working_directory"].(string)

		// requested should be relative to the output folder
		requestedWorkingDirectory = filepath.Join(workingDirectory, requestedWorkingDirectory)

		if _, err := os.Stat(requestedWorkingDirectory); os.IsNotExist(err) {
			return "", fmt.Errorf("working directory does not exist")
		}

		workingDirectory = requestedWorkingDirectory
	}

	slog.Info("Asking permission to run command", "command", command, "working_directory", workingDirectory)

	reader := bufio.NewReader(os.Stdin)
	for {
		slog.Info("Type 'YES' to run the command or 'NO' to skip:")
		input, _ := reader.ReadString('\n') //nolint:errcheck
		input = strings.TrimSpace(input)

		switch input {
		case "YES":
			slog.Info("Running command", "command", command, "working_directory", workingDirectory)

			c := strings.Split(command, " ")

			cmd := exec.Command(c[0], c[1:]...)
			cmd.Dir = workingDirectory
			output, err := cmd.Output()
			if err != nil {
				return "", fmt.Errorf("error running command: %w", err)
			}

			slog.Info("Command output", "output", string(output))

			return string(output), nil
		case "NO":
			return "", fmt.Errorf("command execution skipped by user")
		default:
			slog.Info("Invalid input. Please type 'YES' or 'NO'")
		}
	}
}

func (t *ToolCaller) writeFile(args map[string]any) (string, error) {
	workingDirectory := t.TaskScheduler.OutputFolder

	if args["file"] == nil {
		return "", fmt.Errorf("file is required")
	}

	file := args["file"].(string)

	if args["content"] == nil {
		return "", fmt.Errorf("content is required")
	}

	content := args["content"].(string)

	filePath := filepath.Join(workingDirectory, file)

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return "", fmt.Errorf("error writing file: %w", err)
	}

	return "", nil
}

func (t *ToolCaller) readFile(args map[string]any) (string, error) {
	workingDirectory := t.TaskScheduler.OutputFolder

	if args["file"] == nil {
		return "", fmt.Errorf("file is required")
	}

	file := args["file"].(string)

	filePath := filepath.Join(workingDirectory, file)

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	return string(content), nil
}

func (t *ToolCaller) listFiles(args map[string]any) (string, error) {
	workingDirectory := t.TaskScheduler.OutputFolder

	if args["working_directory"] != nil {
		requestedWorkingDirectory := args["working_directory"].(string)

		// requested should be relative to the output folder
		requestedWorkingDirectory = filepath.Join(workingDirectory, requestedWorkingDirectory)

		if _, err := os.Stat(requestedWorkingDirectory); os.IsNotExist(err) {
			return "", fmt.Errorf("working directory does not exist")
		}

		workingDirectory = requestedWorkingDirectory
	}

	files, err := os.ReadDir(workingDirectory)
	if err != nil {
		return "", fmt.Errorf("error listing files: %w", err)
	}

	filesList := []string{}
	for _, file := range files {
		filesList = append(filesList, file.Name())
	}

	return strings.Join(filesList, "\n"), nil
}

func (t *ToolCaller) editFile(args map[string]any) (string, error) {
	workingDirectory := t.TaskScheduler.OutputFolder

	if args["file"] == nil {
		return "", fmt.Errorf("file is required")
	}

	file := args["file"].(string)

	if args["content"] == nil {
		return "", fmt.Errorf("content is required")
	}

	content := args["content"].(string)

	filePath := filepath.Join(workingDirectory, file)

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return "", fmt.Errorf("error writing file: %w", err)
	}

	return "", nil
}
