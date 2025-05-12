package spawner

import (
	"context"
	"net/url"
)

type Spawner interface {
	Spawn(ctx context.Context) error
	Stop() error
	GetUrl() *url.URL
}

func NewSpawner(engine string) Spawner {
	switch engine {
	case "ollama":
		return NewOllamaSpawner()
	default:
		return nil
	}
}
