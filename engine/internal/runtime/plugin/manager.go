package plugin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

type Request struct {
	Method string         `json:"method"`
	Params map[string]any `json:"params"`
	ID     int            `json:"id"`
}

type Response struct {
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
	ID     int    `json:"id"`
}

type PluginProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	mu     sync.Mutex
	nextID int
}

type Manager struct {
	plugins map[string]*PluginProcess
	mu      sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		plugins: make(map[string]*PluginProcess),
	}
}

func (m *Manager) StartPlugin(name string, command string, args ...string) error {
	cmd := exec.Command(command, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start plugin %s: %w", name, err)
	}

	m.mu.Lock()
	m.plugins[name] = &PluginProcess{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewScanner(stdoutPipe),
	}
	m.mu.Unlock()

	return nil
}

func (m *Manager) Execute(ctx context.Context, pluginName string, script string, params map[string]any) (any, error) {
	m.mu.RLock()
	p, ok := m.plugins[pluginName]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("plugin %q not running", pluginName)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.nextID++
	req := Request{
		Method: "execute",
		Params: map[string]any{
			"script": script,
			"params": params,
		},
		ID: p.nextID,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	reqBytes = append(reqBytes, '\n')

	if _, err := p.stdin.Write(reqBytes); err != nil {
		return nil, fmt.Errorf("failed to write to plugin: %w", err)
	}

	if !p.stdout.Scan() {
		return nil, fmt.Errorf("plugin %q did not respond", pluginName)
	}

	var resp Response
	if err := json.Unmarshal(p.stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse plugin response: %w", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("plugin error: %s", resp.Error)
	}

	return resp.Result, nil
}

func (m *Manager) StopPlugin(name string) error {
	m.mu.Lock()
	p, ok := m.plugins[name]
	if ok {
		delete(m.plugins, name)
	}
	m.mu.Unlock()

	if !ok {
		return nil
	}

	p.stdin.Close()

	done := make(chan error, 1)
	go func() { done <- p.cmd.Wait() }()

	select {
	case err := <-done:
		return err
	case <-time.After(3 * time.Second):
		if p.cmd.Process != nil {
			p.cmd.Process.Kill()
		}
		return nil
	}
}

func (m *Manager) StopAll() {
	m.mu.Lock()
	names := make([]string, 0, len(m.plugins))
	for name := range m.plugins {
		names = append(names, name)
	}
	m.mu.Unlock()

	for _, name := range names {
		m.StopPlugin(name)
	}
}

func (m *Manager) Run(ctx context.Context, script string, params map[string]any) (any, error) {
	runtime := m.detectRuntime(script)
	return m.Execute(ctx, runtime, script, params)
}

func (m *Manager) detectRuntime(script string) string {
	if len(script) > 3 {
		ext := script[len(script)-3:]
		if ext == ".py" {
			return "python"
		}
	}
	return "typescript"
}

func (m *Manager) StartTypescript(nodeCmd string) error {
	if nodeCmd == "" {
		nodeCmd = "node"
	}
	return m.StartPlugin("typescript", nodeCmd, "plugins/typescript/index.js")
}

func (m *Manager) StartPython(pythonCmd string) error {
	if pythonCmd == "" {
		pythonCmd = "python3"
	}
	return m.StartPlugin("python", pythonCmd, "plugins/python/runtime.py")
}

func (m *Manager) IsRunning(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.plugins[name]
	return ok
}
