package config

type Config struct {
	Engine string  `yaml:"engine"`
	Goal   string  `yaml:"goal"`
	Agents []Agent `yaml:"agents"`
}

type Agent struct {
	Name   string `yaml:"name"`
	Model  string `yaml:"model"`
	Prompt string `yaml:"prompt"`
	Tools  []Tool `yaml:"tools"`
}

type Tool struct {
	Type     string    `yaml:"type"`
	Function *Function `yaml:"function,omitempty"`
}

type Function struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Parameters  Parameter `yaml:"parameters"`
}

type Parameter struct {
	Type       string                         `yaml:"type"`
	Required   []string                       `yaml:"required"`
	Properties map[string]ParameterProperties `yaml:"properties"`
}

type ParameterProperties struct {
	Type        string `yaml:"type"`
	Items       any    `yaml:"items,omitempty"`
	Description string `yaml:"description"`
	Enum        []any  `yaml:"enum,omitempty"`
}
