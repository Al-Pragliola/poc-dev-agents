package spawner

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/ollama/ollama/api"
)

type OllamaSpawner struct {
	Url url.URL
	cmd *exec.Cmd
}

func NewOllamaSpawner() *OllamaSpawner {
	return &OllamaSpawner{}
}

func (s *OllamaSpawner) Spawn(ctx context.Context) error {
	port, err := s.findFreePort()
	if err != nil {
		return fmt.Errorf("failed to find free port: %w", err)
	}

	s.Url = url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("127.0.0.1:%d", port),
	}

	slog.Info("Ollama will be available at", "url", s.Url.String())

	s.cmd = exec.Command("ollama", "serve")

	// Create pipes for stdout and stderr
	stdout, err := s.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := s.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start goroutines to read from pipes
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			slog.Debug("ollama stdout", "line", scanner.Text())
		}
	}()
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			slog.Debug("ollama stderr", "line", scanner.Text())
		}
	}()

	s.cmd.Env = os.Environ()
	s.cmd.Env = append(s.cmd.Env, fmt.Sprintf("OLLAMA_HOST=127.0.0.1:%d", port))

	// Start the server process
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ollama: %w", err)
	}

	// Create channels for process monitoring and connection status
	processDone := make(chan error, 1)
	serverReady := make(chan struct{})

	// Monitor the process
	go func() {
		processDone <- s.cmd.Wait()
	}()

	// Check for server readiness
	go func() {
		client := api.NewClient(&s.Url, &http.Client{Timeout: time.Second})

		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := client.Heartbeat(ctx); err == nil {
					close(serverReady)
					return
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	// Wait for either server ready or process failure
	select {
	case <-serverReady:
		slog.Info("Ollama server is ready")
		return nil
	case err := <-processDone:
		return fmt.Errorf("ollama process failed: %w", err)
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout waiting for ollama server to start")
	}
}

func (s *OllamaSpawner) GetUrl() *url.URL {
	return &s.Url
}

func (s *OllamaSpawner) Stop() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return nil
	}

	if err := s.cmd.Process.Signal(os.Interrupt); err != nil {
		// If interrupt fails, try to kill the process
		return s.cmd.Process.Kill()
	}

	return nil
}

func (s *OllamaSpawner) findFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}
