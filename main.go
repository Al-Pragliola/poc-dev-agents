package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Al-Pragliola/poc-dev-agents/internal/agent"
	"github.com/Al-Pragliola/poc-dev-agents/internal/config"
	"github.com/Al-Pragliola/poc-dev-agents/internal/scheduler"
	"gopkg.in/yaml.v3"
)

func main() {
	var config config.Config
	agents := make(map[string]*agent.Agent)

	configFile := flag.String("config", "config.yaml", "The path to the config file")
	outputFolder := flag.String("output", "output", "The path to the output folder")

	flag.Parse()

	yamlFile, err := os.ReadFile(*configFile)
	if err != nil {
		slog.Error("Error reading config file:", "error", err)

		return
	}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		slog.Error("Error unmarshalling config file:", "error", err)

		return
	}

	slog.Info("The goal for the project is: ", "goal", config.Goal)

	err = os.MkdirAll(*outputFolder, 0755)
	if err != nil {
		slog.Error("Error creating output folder:", "error", err)

		return
	}

	for _, a := range config.Agents {
		agents[a.Name] = agent.NewAgent(config.Engine, a)

		if err := agents[a.Name].Setup(); err != nil {
			slog.Error("Error setting up agent:", "error", err)

			return
		}
	}

	defer func() {
		for _, a := range agents {
			if err := a.Teardown(); err != nil {
				slog.Error("Error tearing down agent:", "error", err)
			}
		}
	}()

	resp, err := agents["project-manager"].Chat(context.Background(), config.Goal)
	if err != nil {
		slog.Error("Error:", "error", err)

		return
	}

	if resp.Message != "" {
		slog.Debug("The response from the project manager is: ", "response", resp.Message)
	}

	if len(resp.ToolsCalls) > 0 {
		slog.Debug("The tools calls from the project manager are: ", "toolsCalls", resp.ToolsCalls)
	}

	taskScheduler := scheduler.NewTaskScheduler(agents, *outputFolder)

	for _, toolCall := range resp.ToolsCalls {
		_, err := taskScheduler.ToolCaller.Call(toolCall.Function.Name, toolCall.Function.Arguments)
		if err != nil {
			slog.Error("Error calling tool:", "error", err)
			return
		}
	}

	// Create a channel to listen for OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the task scheduler in a goroutine
	go taskScheduler.Run()

	// Wait for signal
	<-sigChan
	slog.Info("Shutting down...")
	taskScheduler.Stop()
}
