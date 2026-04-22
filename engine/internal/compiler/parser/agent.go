package parser

import (
	"encoding/json"
	"fmt"
	"os"
)

type TriggerDefinition struct {
	Event  string `json:"event"`
	Action string `json:"action"`
	Script string `json:"script"`
}

type CronDefinition struct {
	Schedule string `json:"schedule"`
	Action   string `json:"action"`
	Script   string `json:"script"`
}

type RetryConfig struct {
	Max     int    `json:"max"`
	Backoff string `json:"backoff,omitempty"`
}

type AgentDefinition struct {
	Name     string              `json:"name"`
	Triggers []TriggerDefinition `json:"triggers,omitempty"`
	Cron     []CronDefinition    `json:"cron,omitempty"`
	Retry    RetryConfig         `json:"retry,omitempty"`
}

func ParseAgent(data []byte) (*AgentDefinition, error) {
	var agent AgentDefinition
	if err := json.Unmarshal(data, &agent); err != nil {
		return nil, fmt.Errorf("invalid agent JSON: %w", err)
	}
	if agent.Name == "" {
		return nil, fmt.Errorf("agent name is required")
	}
	if len(agent.Triggers) == 0 && len(agent.Cron) == 0 {
		return nil, fmt.Errorf("agent must have at least one trigger or cron job")
	}
	return &agent, nil
}

func ParseAgentFile(path string) (*AgentDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read agent file %s: %w", path, err)
	}
	return ParseAgent(data)
}
