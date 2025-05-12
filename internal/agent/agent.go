package agent

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Al-Pragliola/poc-dev-agents/internal/config"
	"github.com/Al-Pragliola/poc-dev-agents/internal/mapper"
	"github.com/Al-Pragliola/poc-dev-agents/internal/spawner"
	"github.com/ollama/ollama/api"
)

type ChatResponse struct {
	Message    string         `json:"message"`
	ToolsCalls []api.ToolCall `json:"tools_calls"`
}

type Agent struct {
	Spawner         spawner.Spawner
	Engine          string
	Config          config.Agent
	MessagesHistory []api.Message
	Client          *api.Client
	Tools           []api.Tool
}

func NewAgent(engine string, config config.Agent) *Agent {

	return &Agent{
		Engine: engine,
		Config: config,
		Tools:  mapper.MapConfigToolsToOllamaTools(config.Tools),
	}
}

func (a *Agent) Setup() error {
	a.MessagesHistory = []api.Message{
		{
			Role:    "system",
			Content: a.Config.Prompt,
		},
	}

	spawner := spawner.NewSpawner(a.Engine)

	slog.Info("Spawning agent", "engine", a.Engine, "model", a.Config.Model, "agent", a.Config.Name)

	if err := spawner.Spawn(context.Background()); err != nil {
		return err
	}

	a.Spawner = spawner

	a.Client = api.NewClient(a.Spawner.GetUrl(), http.DefaultClient)

	return nil
}

func (a *Agent) Chat(ctx context.Context, message string) (ChatResponse, error) {
	chatResponse := ChatResponse{
		ToolsCalls: []api.ToolCall{},
	}

	a.MessagesHistory = append(a.MessagesHistory, api.Message{
		Role:    "user",
		Content: message,
	})

	var fullResponse strings.Builder
	err := a.Client.Chat(ctx, &api.ChatRequest{
		Model:    a.Config.Model,
		Messages: a.MessagesHistory,
		Tools:    a.Tools,
	}, func(response api.ChatResponse) error {
		if len(response.Message.ToolCalls) > 0 {
			chatResponse.ToolsCalls = append(chatResponse.ToolsCalls, response.Message.ToolCalls...)
		}

		fullResponse.WriteString(response.Message.Content)
		return nil
	})

	if err != nil {
		return chatResponse, fmt.Errorf("chat error: %w", err)
	}

	// Add the complete response to message history
	a.MessagesHistory = append(a.MessagesHistory, api.Message{
		Role:    "assistant",
		Content: fullResponse.String(),
	})

	chatResponse.Message = fullResponse.String()

	return chatResponse, nil
}

func (a *Agent) Teardown() error {
	return a.Spawner.Stop()
}
