package mapper

import (
	"github.com/Al-Pragliola/poc-dev-agents/internal/config"
	"github.com/ollama/ollama/api"
)

func MapConfigToolsToOllamaTools(t []config.Tool) []api.Tool {
	tools := make([]api.Tool, 0)

	for _, tool := range t {
		properties := make(map[string]struct {
			Type        api.PropertyType `json:"type"`
			Items       any              `json:"items,omitempty"`
			Description string           `json:"description"`
			Enum        []any            `json:"enum,omitempty"`
		})

		for name, property := range tool.Function.Parameters.Properties {
			properties[name] = struct {
				Type        api.PropertyType `json:"type"`
				Items       any              `json:"items,omitempty"`
				Description string           `json:"description"`
				Enum        []any            `json:"enum,omitempty"`
			}{
				Type:        api.PropertyType{property.Type},
				Items:       property.Items,
				Description: property.Description,
				Enum:        property.Enum,
			}
		}

		tools = append(tools, api.Tool{
			Type: tool.Type,
			Function: api.ToolFunction{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters: struct {
					Type       string   `json:"type"`
					Defs       any      `json:"$defs,omitempty"`
					Items      any      `json:"items,omitempty"`
					Required   []string `json:"required"`
					Properties map[string]struct {
						Type        api.PropertyType `json:"type"`
						Items       any              `json:"items,omitempty"`
						Description string           `json:"description"`
						Enum        []any            `json:"enum,omitempty"`
					} `json:"properties"`
				}{
					Type:       tool.Function.Parameters.Type,
					Properties: properties,
					Required:   tool.Function.Parameters.Required,
				},
			},
		})
	}

	return tools
}
